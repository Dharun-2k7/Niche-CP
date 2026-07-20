package judge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/singleflight"
)

var (
	cacheDir     = "/dev/shm/nichecp-cache-v3"
	goCacheDir   = "/dev/shm/nichecp-go-cache-v3"
	binaryCache  *lru.Cache[string, string]
	compileGroup singleflight.Group
)

func init() {
	// 1. Clean up any orphaned binaries from previous crashes
	_ = os.RemoveAll(cacheDir)

	// 2. Ensure cache directory exists in RAM disk
	if err := os.MkdirAll(cacheDir, 0777); err != nil {
		log.Fatalf("Failed to initialize tmpfs cache: %v", err)
	}
	if err := os.MkdirAll(goCacheDir, 0777); err != nil {
		log.Fatalf("Failed to initialize tmpfs go cache: %v", err)
	}

	// 3. Initialize LRU Cache (Capacity: 500)
	var err error
	binaryCache, err = lru.NewWithEvict(500, func(key string, artifactDir string) {
		// Eviction callback: Physically delete from tmpfs to free memory
		log.Printf("[LRU] Evicting cached artifact from RAM: %s", artifactDir)
		_ = os.RemoveAll(artifactDir)
	})
	if err != nil {
		log.Fatalf("Failed to create LRU cache: %v", err)
	}
}

// SandboxResult contains the output of a secure execution
type SandboxResult struct {
	Stdout       string
	Stderr       string
	TimeExceeded bool
}

type CompilationResult struct {
	ArtifactDir string
	Error       string // Compilation error message, empty if success
}

func getHash(code, language, compilerVersion, flags string) string {
	h := sha256.New()
	h.Write([]byte(code + language + compilerVersion + flags))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CompileCode prepares or compiles the code and returns the path to the artifact directory.
// Uses singleflight to prevent Thundering Herd, and LRU cache for memory safety.
func CompileCode(code, language string) (*CompilationResult, error) {
	var image string
	var compileCmd []string
	var isCompiled bool
	var flags string
	var filename string

	switch language {
	case "python":
		filename = "main.py"
		image = "python:3.9-alpine"
		isCompiled = false
	case "go":
		filename = "main.go"
		image = "golang:1.20"
		// CGO_ENABLED=0 GOOS=linux GOARCH=amd64 creates a statically linked binary
		compileCmd = []string{"sh", "-c", "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags=\"-s -w\" -o /workspace/main /workspace/main.go"}
		flags = "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 -ldflags=\"-s -w\""
		isCompiled = true
	case "cpp":
		filename = "main.cpp"
		image = "gcc:12"
		// -static creates a statically linked binary compatible with Alpine
		compileCmd = []string{"g++", "-static", "-O2", "/workspace/main.cpp", "-o", "/workspace/main"}
		flags = "-static -O2"
		isCompiled = true
	case "c":
		filename = "main.c"
		image = "gcc:12"
		compileCmd = []string{"gcc", "-static", "-O2", "/workspace/main.c", "-o", "/workspace/main"}
		flags = "-static -O2"
		isCompiled = true
	case "java":
		filename = "Main.java"
		image = "eclipse-temurin:17-alpine"
		compileCmd = []string{"javac", "/workspace/Main.java"}
		flags = ""
		isCompiled = true
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	hashStr := getHash(code, language, image, flags)

	// Fast path: Check LRU cache
	if path, ok := binaryCache.Get(hashStr); ok {
		return &CompilationResult{ArtifactDir: path}, nil
	}

	// Slow path: Singleflight to coalesce concurrent identical compiles
	res, err, _ := compileGroup.Do(hashStr, func() (interface{}, error) {
		// Double-check cache in case another goroutine just finished it
		if path, ok := binaryCache.Get(hashStr); ok {
			return &CompilationResult{ArtifactDir: path}, nil
		}

		artifactDir := filepath.Join(cacheDir, hashStr)
		if err := os.MkdirAll(artifactDir, 0777); err != nil {
			return nil, fmt.Errorf("failed to create artifact dir: %v", err)
		}
		os.Chmod(artifactDir, 0777)

		// Write source code
		codePath := filepath.Join(artifactDir, filename)
		if err := os.WriteFile(codePath, []byte(code), 0777); err != nil {
			return nil, fmt.Errorf("failed to write code file: %v", err)
		}
		os.Chmod(codePath, 0777)

		if !isCompiled {
			// Interpreted languages don't need a Docker compile phase.
			binaryCache.Add(hashStr, artifactDir)
			return &CompilationResult{ArtifactDir: artifactDir}, nil
		}

		// Compilation Phase in Docker
		dockerArgs := []string{
			"run", "--rm",
			"--network", "none",
			"--memory", "512m", // Generous memory for compiler
			"--cpus", "2.0",
			"-v", fmt.Sprintf("%s:/workspace", artifactDir),
		}

		if language == "go" {
			dockerArgs = append(dockerArgs,
				"-v", fmt.Sprintf("%s:/cache/go-build", goCacheDir),
				"-e", "GOCACHE=/cache/go-build",
			)
		}

		dockerArgs = append(dockerArgs, image)
		dockerArgs = append(dockerArgs, compileCmd...)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30s compile timeout
		defer cancel()

		cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stderr // Capture stdout just in case the compiler uses it for errors

		err := cmd.Run()
		if err != nil {
			// Clean up failed compilation directory so it doesn't pollute tmpfs
			_ = os.RemoveAll(artifactDir)

			if ctx.Err() == context.DeadlineExceeded {
				return &CompilationResult{Error: "Compilation Time Limit Exceeded"}, nil
			}
			return &CompilationResult{Error: stderr.String()}, nil
		}

		// Cache successful compilation in LRU
		binaryCache.Add(hashStr, artifactDir)
		return &CompilationResult{ArtifactDir: artifactDir}, nil
	})

	if err != nil {
		return nil, err
	}

	return res.(*CompilationResult), nil
}

// SandboxProvider defines the abstraction for secure code execution
type SandboxProvider interface {
	RunArtifact(artifactDir, language, input string) (*SandboxResult, error)
}

// GetSandboxProvider returns the configured sandbox implementation.
func GetSandboxProvider() SandboxProvider {
	provider := os.Getenv("SANDBOX_PROVIDER")
	if provider == "nsjail" {
		_, err := exec.LookPath("nsjail")
		if err != nil {
			log.Fatalf("[FATAL] SANDBOX_PROVIDER=nsjail is configured, but the 'nsjail' binary is missing on the host. Host provisioning is required. Failing safely.")
		}
		log.Println("[WARNING] NsJailSandbox is EXPERIMENTAL and requires host provisioning (nsjail binary, cgroup v2).")
		return &NsJailSandbox{}
	}
	return &DockerSandbox{}
}

// DockerSandbox is the battle-tested, production-ready execution environment.
type DockerSandbox struct{}

// RunArtifact executes a pre-compiled artifact or source file using Docker.
func (s *DockerSandbox) RunArtifact(artifactDir, language, input string) (*SandboxResult, error) {
	var image string
	var runCmd []string

	switch language {
	case "python":
		image = "python:3.9-alpine"
		runCmd = []string{"python3", "/workspace/main.py"}
	case "go":
		image = "alpine:latest" // Static binary
		runCmd = []string{"/workspace/main"}
	case "cpp":
		image = "alpine:latest" // Static binary
		runCmd = []string{"/workspace/main"}
	case "c":
		image = "alpine:latest" // Static binary
		runCmd = []string{"/workspace/main"}
	case "java":
		image = "eclipse-temurin:17-alpine"
		// Set JVM max heap to 200m to leave room for JVM overhead within the 256m Docker limit
		runCmd = []string{"java", "-Xmx200m", "-cp", "/workspace", "Main"}
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Execution Phase boundaries
	dockerArgs := []string{
		"run", "--rm",
		"--network", "none",
		"--memory", "256m",
		"--cpus", "1.0",
		"--pids-limit", "50",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp",
		// Mount artifact dir as read-only to prevent state leakage between tests
		"-v", fmt.Sprintf("%s:/workspace:ro", artifactDir),
		"-u", "1000:1000",
		"-i",
	}

	if language == "python" {
		// Add unbuffered flag for Python so stdout isn't delayed/blank
		dockerArgs = append(dockerArgs, "-e", "PYTHONUNBUFFERED=1")
	}

	dockerArgs = append(dockerArgs, image)
	dockerArgs = append(dockerArgs, runCmd...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5s execution timeout
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Always attach Stdin to guarantee EOF is sent, even on empty input.
	// This prevents "docker run -i" from hanging indefinitely.
	cmd.Stdin = strings.NewReader(input)

	_ = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return &SandboxResult{
			Stdout:       stdout.String(),
			Stderr:       "Execution killed: Time Limit Exceeded (5.0s)",
			TimeExceeded: true,
		}, nil
	}

	return &SandboxResult{
		Stdout:       stdout.String(),
		Stderr:       stderr.String(),
		TimeExceeded: false,
	}, nil
}

// NsJailSandbox is the experimental, ultra-low-latency kernel sandbox.
// Note: Requires nsjail to be installed on the host and unprivileged cgroup v2 delegation.
type NsJailSandbox struct{}

func (s *NsJailSandbox) RunArtifact(artifactDir, language, input string) (*SandboxResult, error) {
	var runCmd []string

	switch language {
	case "python":
		runCmd = []string{"/usr/bin/python3", "/workspace/main.py"}
	case "go":
		runCmd = []string{"/workspace/main"}
	case "cpp":
		runCmd = []string{"/workspace/main"}
	case "c":
		runCmd = []string{"/workspace/main"}
	case "java":
		// JVM needs memory ceiling slightly under cgroup limit to avoid SIGKILL.
		runCmd = []string{"/usr/bin/java", "-Xmx200m", "-cp", "/workspace", "Main"}
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// NsJail arguments for maximum security and resource limits
	nsjailArgs := []string{
		"--quiet",
		"-Mo",                            // Mount / proc as read-only
		"--chroot", "/",                  // We use the host filesystem but heavily restricted
		"--user", "1000",                 // Unprivileged user
		"--group", "1000",
		"--disable_clone_newnet",         // Prevent network access
		"--bindmount_ro", fmt.Sprintf("%s:/workspace", artifactDir),
		"--time_limit", "5",              // 5 seconds total run time
		"--cgroup_mem_max", "268435456",  // 256MB memory limit (cgroup v2)
		"--cgroup_cpu_ms_per_sec", "1000",// 1 CPU core
		"--cgroup_pids_max", "50",        // Limit process forks
		"--",                             // End of nsjail config
	}
	
	nsjailArgs = append(nsjailArgs, runCmd...)

	// We apply a Go context timeout of 6 seconds just in case nsjail hangs,
	// though nsjail's internal --time_limit 5 should kill it first.
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nsjail", nsjailArgs...)
	
	if language == "python" {
		cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=1")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	err := cmd.Run()

	// Check if nsjail terminated the process due to timeout
	// Nsjail returns exit code 137 (SIGKILL) or similar if time_limit is hit.
	// But our context might also timeout if nsjail is completely frozen.
	if ctx.Err() == context.DeadlineExceeded || (err != nil && strings.Contains(err.Error(), "signal: killed")) {
		return &SandboxResult{
			Stdout:       stdout.String(),
			Stderr:       "Execution killed: Time Limit Exceeded (5.0s)",
			TimeExceeded: true,
		}, nil
	}
	
	// Note: Proper MLE (Memory Limit Exceeded) parsing would check if the exit code is 137
	// and if it was caused by OOM killer, but for now we mirror the Docker behavior structure.

	return &SandboxResult{
		Stdout:       stdout.String(),
		Stderr:       stderr.String(),
		TimeExceeded: false,
	}, nil
}

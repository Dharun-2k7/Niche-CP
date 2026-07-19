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
	cacheDir     = "/dev/shm/nichecp-cache"
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
		// CGO_ENABLED=0 creates a statically linked binary
		compileCmd = []string{"sh", "-c", "CGO_ENABLED=0 go build -a -installsuffix cgo -o /workspace/main /workspace/main.go"}
		flags = "CGO_ENABLED=0 -a -installsuffix cgo"
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
		image = "openjdk:17-alpine"
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

		// Write source code
		codePath := filepath.Join(artifactDir, filename)
		if err := os.WriteFile(codePath, []byte(code), 0777); err != nil {
			return nil, fmt.Errorf("failed to write code file: %v", err)
		}

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
			image,
		}
		dockerArgs = append(dockerArgs, compileCmd...)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10s compile timeout
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

// RunArtifact executes a pre-compiled artifact or source file for the given language.
// Uses ultra-lightweight alpine containers for statically compiled languages.
func RunArtifact(artifactDir, language, input string) (*SandboxResult, error) {
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
		image = "openjdk:17-alpine"
		runCmd = []string{"java", "-cp", "/workspace", "Main"}
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
		"-i",
		image,
	}
	dockerArgs = append(dockerArgs, runCmd...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5s execution timeout
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

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

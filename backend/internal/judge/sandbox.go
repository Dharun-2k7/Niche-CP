package judge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SandboxResult contains the output of a secure execution
type SandboxResult struct {
	Stdout       string
	Stderr       string
	TimeExceeded bool
}

// RunSecurely takes raw code and input, runs it in an isolated Docker container, and returns the result
func RunSecurely(code, language, input string) (*SandboxResult, error) {
	// 1. Create a secure temporary directory
	tempDir, err := os.MkdirTemp("", "judge_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Ensure cleanup happens

	// 2. Setup file extensions and execution commands based on language
	var filename string
	var image string
	var runCmd []string

	switch language {
	case "python":
		filename = "main.py"
		image = "python:3.9-slim"
		runCmd = []string{"python3", "/workspace/main.py"}
	case "go":
		filename = "main.go"
		image = "golang:1.20"
		// 'go run' inside docker is slow, but fine for MVP
		runCmd = []string{"go", "run", "/workspace/main.go"}
	case "cpp":
		filename = "main.cpp"
		image = "gcc:12"
		// Needs compile step first, then run
		runCmd = []string{"sh", "-c", "g++ /workspace/main.cpp -o /workspace/main && /workspace/main"}
	case "c":
		filename = "main.c"
		image = "gcc:12"
		runCmd = []string{"sh", "-c", "gcc /workspace/main.c -o /workspace/main && /workspace/main"}
	case "java":
		filename = "Main.java"
		image = "openjdk:17-jdk-slim"
		runCmd = []string{"sh", "-c", "javac /workspace/Main.java && java -cp /workspace Main"}
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	codePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code file: %v", err)
	}

	// 3. Construct Docker command with security boundaries
	// --rm: remove container after exit
	// --network none: Disable internet access
	// --memory="256m": Prevent memory bombs
	// --cpus="1.0": Prevent CPU exhaustion
	dockerArgs := []string{
		"run", "--rm",
		"--network", "none",
		"--memory", "256m",
		"--cpus", "1.0",
		"-v", fmt.Sprintf("%s:/workspace", tempDir),
		"-i", // Keep STDIN open even if not attached
		image,
	}
	dockerArgs = append(dockerArgs, runCmd...)

	// 4. Setup Execution with Timeout (5.0 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Pass custom input to STDIN of the docker container
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	// 5. Run the container
	err = cmd.Run()

	// Check if it was killed by our Context Timeout
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

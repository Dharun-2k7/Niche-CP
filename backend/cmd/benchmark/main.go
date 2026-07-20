package main

import (
	"fmt"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
)

func main() {
	goCode1 := `
package main
import "fmt"
func main() {
	var n int
	fmt.Scan(&n)
	fmt.Println(n * 2)
}
`

	goCode2 := `
package main
import "fmt"
func main() {
	var n int
	fmt.Scan(&n)
	fmt.Println(n * 3) // Different code
}
`

	fmt.Println("--- AFTER BENCHMARK (GO CACHE) ---")

	// Case 1: Cold Compile
	fmt.Println("\n[Case 1: Cold Go Compile]")
	startCompile1 := time.Now()
	goComp1, err := judge.CompileCode(goCode1, "go")
	if err != nil || goComp1.Error != "" {
		fmt.Printf("Go Compile Error: %v\n%s\n", err, goComp1.Error)
		return
	}
	compileDur1 := time.Since(startCompile1)
	fmt.Printf("Cold Compile: %s\n", compileDur1)

	startExec1 := time.Now()
	goRes1, err := judge.RunArtifact(goComp1.ArtifactDir, "go", "42\n")
	execDur1 := time.Since(startExec1)
	fmt.Printf("Execution: %s\n", execDur1)
	if goRes1 != nil {
		fmt.Printf("Output: %q\n", goRes1.Stdout)
	}

	// Case 2: Warm Compile (Identical Code - LRU Cache Test)
	fmt.Println("\n[Case 2: Warm Go Compile (Identical Code)]")
	startCompile2 := time.Now()
	goComp2, err := judge.CompileCode(goCode1, "go")
	if err != nil || goComp2.Error != "" {
		fmt.Printf("Go Compile Error: %v\n%s\n", err, goComp2.Error)
		return
	}
	compileDur2 := time.Since(startCompile2)
	fmt.Printf("Warm Compile (LRU): %s\n", compileDur2)
	fmt.Printf("Same ArtifactDir: %v\n", goComp1.ArtifactDir == goComp2.ArtifactDir)

	// Case 3: Warm Compile (Different Code - GOCACHE Test)
	fmt.Println("\n[Case 3: Warm Go Compile (Different Code)]")
	startCompile3 := time.Now()
	goComp3, err := judge.CompileCode(goCode2, "go")
	if err != nil || goComp3.Error != "" {
		fmt.Printf("Go Compile Error: %v\n%s\n", err, goComp3.Error)
		return
	}
	compileDur3 := time.Since(startCompile3)
	fmt.Printf("Warm Compile (GOCACHE): %s\n", compileDur3)

	startExec3 := time.Now()
	goRes3, err := judge.RunArtifact(goComp3.ArtifactDir, "go", "42\n")
	execDur3 := time.Since(startExec3)
	fmt.Printf("Execution: %s\n", execDur3)
	if goRes3 != nil {
		fmt.Printf("Output: %q\n", goRes3.Stdout) // Should be 126 (42 * 3)
	}
	fmt.Printf("Total: %s\n", compileDur3+execDur3)
}

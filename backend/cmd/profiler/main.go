package main

import (
	"log"
	"strings"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
)

// Instrumented RunSecurely
func RunSecurelyProfiled(code, language string, inputs []string) {
	log.Println("--- Compile Phase ---")
	startCompile := time.Now()
	compRes, err := judge.CompileCode(code, language)
	if err != nil || compRes.Error != "" {
		log.Fatalf("Compilation failed: %v, %v", err, compRes.Error)
	}
	log.Printf("Compilation + Cache check took: %v", time.Since(startCompile))

	for i, tcInput := range inputs {
		log.Printf("--- Test Case %d ---", i+1)
		
		startDocker := time.Now()
		provider := judge.GetSandboxProvider()
		session, err := provider.StartSession(compRes.ArtifactDir, language)
		if err != nil {
			log.Fatalf("Failed to start session: %v", err)
		}
		res, err := session.RunTestcase(tcInput)
		session.Close()
		log.Printf("Docker execution (Startup + Run) took: %v", time.Since(startDocker))
		
		if err != nil {
			log.Printf("Execution error: %v", err)
		}
		
		startComparison := time.Now()
		expectedOutput := "mock expected output"
		actualOutput := strings.TrimSpace(res.Stdout)
		_ = actualOutput == expectedOutput
		log.Printf("Output comparison took: %v", time.Since(startComparison))
	}
}

func main() {
	// Initialize judge cache properly
	log.Println("Starting Profiler...")

	code := `#include <iostream>
int main() {
	int a, b;
	std::cin >> a >> b;
	std::cout << (a + b) << std::endl;
	return 0;
}`
	language := "cpp"
	
	inputs := []string{"1 2", "3 4", "5 6"} // 3 mock testcases
	
	startTotal := time.Now()
	RunSecurelyProfiled(code, language, inputs)
	log.Printf("Total end-to-end execution loop for 3 testcases took: %v", time.Since(startTotal))
}

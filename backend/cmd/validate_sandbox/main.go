package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
)

// This script prepares the validation framework for comparing SandboxProviders.
// Currently it tests the default DockerSandbox since NsJail is missing dependencies.

type TestCase struct {
	Language string
	Code     string
	Input    string
	ExpectedVerdict string // "ACCEPTED", "TIME_LIMIT_EXCEEDED", "RUNTIME_ERROR"
}

func main() {
	log.Println("Starting Sandbox Validation Framework...")
	
	// Create SandboxProviders
	dockerSandbox := &judge.DockerSandbox{}
	
	// Ensure cache dirs exist
	_ = os.MkdirAll("/dev/shm/nichecp-cache-v3", 0777)
	
	testCases := []TestCase{
		{
			Language: "cpp",
			Code: `#include <iostream>
using namespace std;
int main() {
	int a;
	cin >> a;
	cout << a * 2;
	return 0;
}`,
			Input: "5",
			ExpectedVerdict: "ACCEPTED",
		},
		{
			Language: "python",
			Code: `while True: pass`,
			Input: "",
			ExpectedVerdict: "TIME_LIMIT_EXCEEDED",
		},
		{
			Language: "go",
			Code: `package main
func main() {
	var a []int
	for {
		a = append(a, 1)
	}
}`,
			Input: "",
			ExpectedVerdict: "RUNTIME_ERROR", // MLE will cause killed signal -> RUNTIME_ERROR
		},
		{
			Language: "java",
			Code: `public class Main {
	public static void main(String[] args) {
		System.out.println("Hello");
	}
}`,
			Input: "",
			ExpectedVerdict: "ACCEPTED",
		},
	}

	for i, tc := range testCases {
		fmt.Printf("\n--- Running Test %d: %s (%s) ---\n", i+1, tc.Language, tc.ExpectedVerdict)
		
		// 1. Compile
		compRes, err := judge.CompileCode(tc.Code, tc.Language)
		if err != nil || compRes.Error != "" {
			fmt.Printf("Compile failed: %v %s\n", err, compRes.Error)
			continue
		}
		
		// 2. Run Docker
		session, err := dockerSandbox.StartSession(compRes.ArtifactDir, tc.Language)
		if err != nil {
			fmt.Printf("Docker Start Error: %v\n", err)
			continue
		}
		res, err := session.RunTestcase(tc.Input)
		session.Close()
		if err != nil {
			fmt.Printf("Docker Execution Error: %v\n", err)
			continue
		}
		
		verdict := "ACCEPTED"
		if res.TimeExceeded {
			verdict = "TIME_LIMIT_EXCEEDED"
		} else if res.Stderr != "" {
			verdict = "RUNTIME_ERROR"
		}
		
		fmt.Printf("Docker Verdict: %s\n", verdict)
		if verdict != tc.ExpectedVerdict {
			fmt.Printf("FAILED: Expected %s but got %s\n", tc.ExpectedVerdict, verdict)
		} else {
			fmt.Println("PASSED!")
		}
	}
	
	fmt.Println("\nValidation suite complete. NsJail validation is pending host provisioning.")
}

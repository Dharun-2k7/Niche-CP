package judge_test

import (
	"testing"

	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
)

func TestCppVerdicts(t *testing.T) {
	// AC Test
	codeAC := `#include <iostream>
using namespace std;
int main() {
    int a, b;
    if (cin >> a >> b) cout << (a + b) << endl;
    return 0;
}`
	comp, err := judge.CompileCode(codeAC, "cpp")
	if err != nil || comp.Error != "" {
		t.Fatalf("C++ AC Compile failed: %v, %s", err, comp.Error)
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(comp.ArtifactDir, "cpp")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}
	defer session.Close()

	res, err := session.RunTestcase("3 5\n")
	if err != nil || res.Stdout != "8\n" {
		t.Errorf("C++ AC test failed. Expected '8\\n', got '%s', err: %v", res.Stdout, err)
	}

	// CE Test
	codeCE := `int main() { invalid_syntax }`
	compCE, _ := judge.CompileCode(codeCE, "cpp")
	if compCE.Error == "" {
		t.Errorf("Expected compilation error for invalid C++ code")
	}
}

func TestPythonVerdicts(t *testing.T) {
	// AC Test
	codeAC := `import sys
data = sys.stdin.read().split()
if len(data) >= 2:
    print(int(data[0]) + int(data[1]))
`
	comp, err := judge.CompileCode(codeAC, "python")
	if err != nil || comp.Error != "" {
		t.Fatalf("Python Compile failed: %v, %s", err, comp.Error)
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(comp.ArtifactDir, "python")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}
	defer session.Close()

	res, err := session.RunTestcase("10 20\n")
	if err != nil || res.Stdout != "30\n" {
		t.Errorf("Python AC test failed. Expected '30\\n', got '%s', err: %v", res.Stdout, err)
	}
}

func TestSecurityIsolation(t *testing.T) {
	// Network access attempt test
	netCode := `import socket
try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(("8.8.8.8", 53))
    print("NET_SUCCESS")
except Exception as e:
    print("NET_BLOCKED")
`
	comp, _ := judge.CompileCode(netCode, "python")
	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(comp.ArtifactDir, "python")
	if err == nil {
		defer session.Close()
		res, _ := session.RunTestcase("")
		if res.Stdout == "NET_SUCCESS\n" {
			t.Errorf("SECURITY VULNERABILITY: Network access allowed in sandbox!")
		}
	}
}

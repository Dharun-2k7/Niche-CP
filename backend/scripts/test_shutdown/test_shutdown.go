package main

import (
	"fmt"
	"os/exec"
	"os"
	"time"
)

func main() {
	cmd := exec.Command("go", "run", "cmd/worker/main.go")
	cmd.Env = append(os.Environ(), "MAX_EXECUTION_WORKERS=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	
	time.Sleep(3 * time.Second)
	fmt.Println("Sending interrupt...")
	cmd.Process.Signal(os.Interrupt)
	cmd.Wait()
	fmt.Println("Process exited!")
}

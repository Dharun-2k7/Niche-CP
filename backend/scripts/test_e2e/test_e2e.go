package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"os"
	"time"
	
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.InitPostgres()
	db.InitRedis()
	
	exec.Command("go", "build", "-o", "worker_bin", "cmd/worker/main.go").Run()
	defer os.Remove("worker_bin")

	fmt.Println("Starting worker process (4 workers)...")
	cmd := exec.Command("./worker_bin")
	cmd.Env = append(os.Environ(), "MAX_EXECUTION_WORKERS=4")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	
	// Wait a moment for workers to connect
	time.Sleep(2 * time.Second)
	
	code := `
import time
time.sleep(1)
print("No")
`
	
	// Ensure Problem 1 Test Cases
	db.DB.Exec(`
		INSERT INTO problems (id, title, hidden_testcases) 
		VALUES (1, 'Test', '[{"input":"1","expected_output":"No"}]') 
		ON CONFLICT (id) DO UPDATE SET hidden_testcases = '[{"input":"1","expected_output":"No"}]'
	`)

	// Prepare DB for 8 submissions
	for i := 0; i < 8; i++ {
		db.DB.Exec(`
			INSERT INTO submissions (id, user_id, problem_id, code, language, status) 
			VALUES ($1, 1, 1, $2, 'python', 'PENDING') 
			ON CONFLICT (id) DO UPDATE SET status = 'PENDING'
		`, i+1000, code)
	}
	
	// Clear the queue first
	db.RedisClient.Del(context.Background(), "submissions_queue")

	fmt.Println("Pushing 8 jobs to queue...")
	start := time.Now()
	
	for i := 0; i < 8; i++ {
		jobData, _ := json.Marshal(map[string]interface{}{
			"submission_id": i + 1000,
			"code":          code,
			"language":      "python",
			"problem_id":    1,
		})
		db.RedisClient.LPush(context.Background(), "submissions_queue", jobData)
	}

	for {
		var pending int
		db.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE status = 'PENDING' AND id >= 1000 AND id <= 1007").Scan(&pending)
		if pending == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	duration := time.Since(start)
	fmt.Printf("\n==================================\n")
	fmt.Printf("=> All 8 jobs finished! Total Time: %s\n", duration)
	fmt.Printf("==================================\n\n")
	
	fmt.Println("Sending interrupt to worker...")
	cmd.Process.Signal(os.Interrupt)
	cmd.Wait() // Wait for graceful shutdown
}

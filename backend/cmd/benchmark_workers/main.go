package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.InitPostgres()
	db.InitRedis()
	
	exec.Command("go", "build", "-o", "worker_bin", "cmd/worker/main.go").Run()

	// Clear queue
	db.RedisClient.Del(context.Background(), "submissions_queue")

	// Ensure problem 1 has test cases
	_, _ = db.DB.Exec(`INSERT INTO problems (id, title, hidden_testcases) VALUES (1, 'Test', '[{"input":"1","expected_output":"1"}]') ON CONFLICT (id) DO UPDATE SET hidden_testcases = '[{"input":"1","expected_output":"1"}]'`)

	// Enqueue 8 Python sleep jobs
	code := `
import time
time.sleep(1)
print("No")
`
	for i := 0; i < 8; i++ {
		_, err := db.DB.Exec(`
			INSERT INTO submissions (id, user_id, problem_id, code, language, status) 
			VALUES ($1, 1, 1, $2, 'python', 'PENDING') 
			ON CONFLICT (id) DO UPDATE SET status = 'PENDING'
		`, i+1000, code)
		if err != nil { fmt.Printf("SQL Error: %v\n", err) }
		
		jobData, _ := json.Marshal(map[string]interface{}{
			"submission_id": i + 1000,
			"code":          code,
			"language":      "python",
			"problem_id":    1,
		})
		db.RedisClient.LPush(context.Background(), "submissions_queue", jobData)
	}

	fmt.Println("Pushed 8 jobs. Benchmarking 2 workers...")
	bench(2)

	// Repush for 4 workers
	for i := 0; i < 8; i++ {
		_, err := db.DB.Exec("UPDATE submissions SET status = 'PENDING' WHERE id = $1", i+1000)
		if err != nil { fmt.Printf("SQL Error 2: %v\n", err) }
		jobData, _ := json.Marshal(map[string]interface{}{
			"submission_id": i + 1000,
			"code":          code,
			"language":      "python",
			"problem_id":    1,
		})
		db.RedisClient.LPush(context.Background(), "submissions_queue", jobData)
	}

	bench(4)
	
	os.Remove("worker_bin")
}

func bench(workers int) {
	cmd := exec.Command("./worker_bin")
	cmd.Env = append(os.Environ(), fmt.Sprintf("MAX_EXECUTION_WORKERS=%d", workers))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	start := time.Now()
	cmd.Start()

	// Wait until queue is empty and jobs processed
	for {
		count, _ := db.RedisClient.LLen(context.Background(), "submissions_queue").Result()
		
		var pending int
		db.DB.QueryRow("SELECT COUNT(*) FROM submissions WHERE status = 'PENDING' AND id >= 1000").Scan(&pending)
		
		if count == 0 && pending == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	
	duration := time.Since(start)
	
	cmd.Process.Signal(os.Interrupt)
	cmd.Wait() // Wait for graceful shutdown

	peak, _ := os.ReadFile("peak_sandboxes.txt")
	fmt.Printf("=> %d Workers took: %s | Peak Sandboxes: %s\n", workers, duration, string(peak))
}

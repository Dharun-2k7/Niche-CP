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
	
	code := `
import time
time.sleep(1)
print("1")
`
	db.DB.Exec(`
			INSERT INTO problems (id, title, hidden_testcases) 
			VALUES (1, 'Test', '[{"input":"1","expected_output":"1"}]') 
			ON CONFLICT (id) DO UPDATE SET hidden_testcases = '[{"input":"1","expected_output":"1"}]'
		`)
	db.DB.Exec(`
			INSERT INTO submissions (id, user_id, problem_id, code, language, status) 
			VALUES ($1, 1, 1, $2, 'python', 'PENDING') 
			ON CONFLICT (id) DO UPDATE SET status = 'PENDING'
		`, 2000, code)
	
	jobData, _ := json.Marshal(map[string]interface{}{
		"submission_id": 2000,
		"code":          code,
		"language":      "python",
		"problem_id":    1,
	})
	db.RedisClient.LPush(context.Background(), "submissions_queue", jobData)
	
	cmd := exec.Command("go", "run", "cmd/worker/main.go")
	cmd.Env = append(os.Environ(), "MAX_EXECUTION_WORKERS=1")
	f, _ := os.Create("worker_logs.txt")
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Start()
	
	// Wait 10 seconds
	time.Sleep(10 * time.Second)
	cmd.Process.Kill()
	fmt.Println("Done")
}

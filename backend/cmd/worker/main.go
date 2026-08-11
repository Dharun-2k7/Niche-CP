package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/joho/godotenv"
)

type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

var (
	activeSandboxes int32
	peakSandboxes   int32
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default environment variables.")
	}

	db.InitPostgres()
	db.InitRedis()
	judge.InitSemaphore()

	maxWorkers := 2 // Default configuration
	if val := os.Getenv("MAX_EXECUTION_WORKERS"); val != "" {
		if w, err := strconv.Atoi(val); err == nil && w > 0 {
			maxWorkers = w
		}
	}

	log.Printf("Worker pool started with %d workers. Listening for submissions...", maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received termination signal. Shutting down worker pool...")
		cancel()
	}()

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerLoop(ctx, workerID)
		}(i + 1)
	}

	wg.Wait()
	log.Printf("Worker pool stopped cleanly. PEAK ACTIVE SANDBOXES: %d", atomic.LoadInt32(&peakSandboxes))

	// Write to file for benchmark script to read
	os.WriteFile("peak_sandboxes.txt", []byte(fmt.Sprintf("%d", atomic.LoadInt32(&peakSandboxes))), 0666)
}

func workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Blocking pop from Redis queues (submissions_queue, run_queue)
		result, err := db.RedisClient.BRPop(ctx, 1*time.Second, "submissions_queue", "run_queue").Result()
		if err != nil {
			if err == context.Canceled {
				return
			}
			if err.Error() == "redis: nil" { // Timeout
				continue
			}
			log.Printf("[Worker %d] Error pulling from queue: %v", workerID, err)
			continue
		}

		queueName := result[0]
		payload := result[1]

		if queueName == "run_queue" {
			processRunJob(ctx, workerID, payload)
		} else {
			processSubmission(ctx, workerID, payload)
		}
	}
}

func processRunJob(ctx context.Context, workerID int, payload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d] Recovered from panic processing run job: %v\nPayload: %s", workerID, r, payload)
		}
	}()

	var job map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[Worker %d] Failed to unmarshal run job payload: %v", workerID, err)
		return
	}

	requestID, okReq := job["request_id"].(string)
	code, okCode := job["code"].(string)
	language, okLang := job["language"].(string)
	input, _ := job["input"].(string)

	if !okReq || !okCode || !okLang {
		log.Printf("[Worker %d] Malformed run job payload: %s", workerID, payload)
		return
	}

	acqCtx, cancelAcq := context.WithTimeout(ctx, 10*time.Second)
	defer cancelAcq()

	err := judge.AcquireExecutionToken(acqCtx)
	if err != nil {
		log.Printf("[Worker %d] Semaphore acquisition failed for run job %s: %v", workerID, requestID, err)
		return
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(code, language)
	if err != nil || compRes.Error != "" {
		errMsg := compRes.Error
		if err != nil {
			errMsg = fmt.Sprintf("Sandbox Error: %v", err)
		}
		resJSON, _ := json.Marshal(map[string]string{"output": "", "stderr": errMsg})
		resKey := fmt.Sprintf("run_result:%s", requestID)
		db.RedisClient.LPush(ctx, resKey, resJSON)
		db.RedisClient.Expire(ctx, resKey, 10*time.Second)
		return
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, language)
	if err != nil {
		resJSON, _ := json.Marshal(map[string]string{"output": "", "stderr": fmt.Sprintf("Sandbox Session Error: %v", err)})
		resKey := fmt.Sprintf("run_result:%s", requestID)
		db.RedisClient.LPush(ctx, resKey, resJSON)
		db.RedisClient.Expire(ctx, resKey, 10*time.Second)
		return
	}
	defer session.Close()

	res, err := session.RunTestcase(input)
	stdout, stderr := "", ""
	if err != nil {
		stderr = fmt.Sprintf("Sandbox Error: %v", err)
	} else {
		stdout = res.Stdout
		stderr = res.Stderr
	}

	resJSON, _ := json.Marshal(map[string]string{"output": stdout, "stderr": stderr})
	resKey := fmt.Sprintf("run_result:%s", requestID)
	db.RedisClient.LPush(ctx, resKey, resJSON)
	db.RedisClient.Expire(ctx, resKey, 10*time.Second)
}

func processSubmission(ctx context.Context, workerID int, payload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d] Recovered from panic processing job: %v\nPayload: %s", workerID, r, payload)
		}
	}()

	var job map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[Worker %d] Failed to unmarshal job payload: %v", workerID, err)
		return
	}

	// Defensive type assertions
	subIDRaw, okSub := job["submission_id"].(float64)
	code, okCode := job["code"].(string)
	language, okLang := job["language"].(string)
	probIDRaw, okProb := job["problem_id"].(float64)

	if !okSub || !okCode || !okLang || !okProb {
		log.Printf("[Worker %d] Malformed job payload missing required fields: %s", workerID, payload)
		if okSub {
			_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", int(subIDRaw))
		}
		return
	}

	submissionID := int(subIDRaw)
	problemID := int(probIDRaw)

	log.Printf("[Worker %d] Processing Submission %d (%s) for Problem %d...", workerID, submissionID, language, problemID)

	// Fetch hidden test cases from Postgres using problemID
	var testCasesJSON string
	err := db.DB.QueryRow("SELECT hidden_testcases FROM problems WHERE id = $1", problemID).Scan(&testCasesJSON)
	if err != nil {
		log.Printf("[Worker %d] Failed to fetch test cases for Problem %d: %v", workerID, problemID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}

	var testCases []TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		log.Printf("[Worker %d] Failed to unmarshal test cases for Problem %d: %v", workerID, problemID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}

	acqCtx, cancelAcq := context.WithTimeout(ctx, 5*time.Minute) // Wait up to 5 mins in queue
	defer cancelAcq()

	err = judge.AcquireExecutionToken(acqCtx)
	if err != nil {
		log.Printf("[Worker %d] Dropping Sub %d due to semaphore acquisition failure: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	
	defer judge.ReleaseExecutionToken() // Guaranteed release even on panic

	// INSTRUMENTATION: Track concurrency
	currentActive := atomic.AddInt32(&activeSandboxes, 1)
	defer atomic.AddInt32(&activeSandboxes, -1)

	for {
		peak := atomic.LoadInt32(&peakSandboxes)
		if currentActive <= peak {
			break
		}
		if atomic.CompareAndSwapInt32(&peakSandboxes, peak, currentActive) {
			break
		}
	}
	log.Printf("[Worker %d] Acquired Token | Current Active Sandboxes: %d | Peak: %d", workerID, currentActive, atomic.LoadInt32(&peakSandboxes))

	// Compile code once before iterating test cases
	compRes, err := judge.CompileCode(code, language)
	if err != nil {
		log.Printf("[Worker %d] Compilation environment error for Sub %d: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	if compRes.Error != "" {
		log.Printf("[Worker %d] Compilation error for Sub %d: %s", workerID, submissionID, compRes.Error)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "COMPILATION_ERROR", submissionID)
		return
	}

	status := "ACCEPTED"
	
	// Fetch the configured Sandbox Provider
	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, language)
	if err != nil {
		log.Printf("[Worker %d] Failed to start sandbox session for Sub %d: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	defer session.Close()

	for i, tc := range testCases {
		res, err := session.RunTestcase(tc.Input)

		if err != nil {
			status = "RUNTIME_ERROR"
			log.Printf("[Worker %d] Execution failed for Sub %d: %v", workerID, submissionID, err)
			break
		} else if res.TimeExceeded {
			status = "TIME_LIMIT_EXCEEDED"
			break
		} else if res.Stderr != "" {
			status = "RUNTIME_ERROR"
			break
		}

		actualOutput := strings.TrimSpace(res.Stdout)
		expectedOutput := strings.TrimSpace(tc.ExpectedOutput)

		log.Printf("[Worker %d] Sub %d Test %d | Expected: %s | Actual: %s", workerID, submissionID, i+1, expectedOutput, actualOutput)
		fmt.Printf("DEBUG: expected='%s', actual='%s'\n", expectedOutput, actualOutput)

		if actualOutput != expectedOutput {
			status = "WRONG_ANSWER"
			log.Printf("[Worker %d] Submission %d failed on test case %d", workerID, submissionID, i+1)
			break
		}
	}

	// Update Database with the verdict
	_, err = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", status, submissionID)
	if err != nil {
		log.Printf("[Worker %d] Failed to update database for Sub %d: %v", workerID, submissionID, err)
	} else {
		log.Printf("[Worker %d] Submission %d completed with status: %s", workerID, submissionID, status)
		if status == "ACCEPTED" {
			var userID int
			err = db.DB.QueryRow("SELECT user_id FROM submissions WHERE id = $1", submissionID).Scan(&userID)
			if err == nil {
				db.DB.Exec(`
					INSERT INTO user_problem_status (user_id, problem_id, status)
					VALUES ($1, $2, $3)
					ON CONFLICT (user_id, problem_id) DO UPDATE SET status = EXCLUDED.status
				`, userID, problemID, status)
			}
		}
	}
}

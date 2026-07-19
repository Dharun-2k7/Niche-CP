package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/joho/godotenv"
	"strings"
)

type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default environment variables.")
	}

	db.InitPostgres()
	db.InitRedis()

	log.Println("Worker started. Listening for submissions...")

	for {
		// BRPop blocks until an item is available in the 'submissions_queue'
		// It prevents high CPU usage compared to a tight polling loop
		result, err := db.RedisClient.BRPop(context.Background(), 0, "submissions_queue").Result()
		if err != nil {
			log.Printf("Error pulling from queue: %v", err)
			time.Sleep(5 * time.Second) // Backoff on error
			continue
		}

		// result[0] is the queue name, result[1] is the JSON payload
		payload := result[1]
		
		// Run job in an anonymous func to safely recover from panics per-job
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Worker recovered from panic processing job: %v\nPayload: %s", r, payload)
				}
			}()

			var job map[string]interface{}
			if err := json.Unmarshal([]byte(payload), &job); err != nil {
				log.Printf("Failed to unmarshal job payload: %v", err)
				return
			}

			// Defensive type assertions
			subIDRaw, okSub := job["submission_id"].(float64)
			code, okCode := job["code"].(string)
			language, okLang := job["language"].(string)
			probIDRaw, okProb := job["problem_id"].(float64)

			if !okSub || !okCode || !okLang || !okProb {
				log.Printf("Malformed job payload missing required fields: %s", payload)
				if okSub {
					_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", int(subIDRaw))
				}
				return
			}

			submissionID := int(subIDRaw)
			problemID := int(probIDRaw)

			log.Printf("Processing Submission %d (%s) for Problem %d...", submissionID, language, problemID)

			// Fetch hidden test cases from Postgres using problemID
			var testCasesJSON string
			err = db.DB.QueryRow("SELECT hidden_testcases FROM problems WHERE id = $1", problemID).Scan(&testCasesJSON)
			if err != nil {
				log.Printf("Failed to fetch test cases for Problem %d: %v", problemID, err)
				_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
				return
			}

			var testCases []TestCase
			if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
				log.Printf("Failed to unmarshal test cases for Problem %d: %v", problemID, err)
				_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
				return
			}

			// Compile code once before iterating test cases
			compRes, err := judge.CompileCode(code, language)
			if err != nil {
				log.Printf("Compilation environment error for Sub %d: %v", submissionID, err)
				_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
				return
			}
			if compRes.Error != "" {
				log.Printf("Compilation error for Sub %d: %s", submissionID, compRes.Error)
				_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "COMPILATION_ERROR", submissionID)
				return
			}

			status := "ACCEPTED"
			for i, tc := range testCases {
				res, err := judge.RunArtifact(compRes.ArtifactDir, language, tc.Input)

				if err != nil {
					status = "RUNTIME_ERROR"
					log.Printf("Docker execution failed for Sub %d: %v", submissionID, err)
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

				if actualOutput != expectedOutput {
					status = "WRONG_ANSWER"
					log.Printf("Submission %d failed on test case %d", submissionID, i+1)
					break
				}
			}

			// Update Database with the verdict
			_, err = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", status, submissionID)
			if err != nil {
				log.Printf("Failed to update database for Sub %d: %v", submissionID, err)
			} else {
				log.Printf("Submission %d completed with status: %s", submissionID, status)
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
		}()
	}
}

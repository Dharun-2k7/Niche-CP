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
		var job map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("Failed to unmarshal job payload: %v", err)
			continue
		}

		submissionID := int(job["submission_id"].(float64))
		code := job["code"].(string)
		language := job["language"].(string)
		problemID := int(job["problem_id"].(float64))

		log.Printf("Processing Submission %d (%s) for Problem %d...", submissionID, language, problemID)

		// Fetch hidden test cases from Postgres using problemID
		var testCasesJSON string
		err = db.DB.QueryRow("SELECT test_cases FROM problems WHERE id = $1", problemID).Scan(&testCasesJSON)
		if err != nil {
			log.Printf("Failed to fetch test cases for Problem %d: %v", problemID, err)
			_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
			continue
		}

		var testCases []TestCase
		if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
			log.Printf("Failed to unmarshal test cases for Problem %d: %v", problemID, err)
			_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
			continue
		}

		status := "ACCEPTED"
		for i, tc := range testCases {
			res, err := judge.RunSecurely(code, language, tc.Input)

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
		}
	}
}

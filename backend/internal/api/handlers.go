package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/gin-gonic/gin"
)

// HealthCheck responds with a simple OK
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "API is healthy!"})
}

// SubmitRequest defines the JSON payload for a code submission
type SubmitRequest struct {
	ProblemID int    `json:"problem_id" binding:"required"`
	ContestID *int   `json:"contest_id"`
	Code      string `json:"code" binding:"required"`
	Language  string `json:"language" binding:"required"`
}

// SubmitCode handles new code submissions
func SubmitCode(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 1. Check Rate Limiting (5 seconds)
	userID := c.GetInt("user_id") // Extracted by auth middleware

	rateLimitKey := fmt.Sprintf("rate_limit:submit:user:%d", userID)
	set, err := db.RedisClient.SetNX(context.Background(), rateLimitKey, "1", 5*time.Second).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
		return
	}
	if !set {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait 5 seconds before submitting again."})
		return
	}

	// 1.5 Validate ProblemID exists to avoid raw 500 DB constraint error
	var exists bool
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM problems WHERE id = $1)", req.ProblemID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// 1.6 Contest Lifecycle Constraints
	if req.ContestID != nil {
		var status string
		var endTime time.Time
		err = db.DB.QueryRow("SELECT status, end_time FROM contests WHERE id = $1", *req.ContestID).Scan(&status, &endTime)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
			return
		}

		if status == "CREATED" || status == "UPCOMING" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet."})
			return
		}

		if status == "RUNNING" {
			var registered bool
			err = db.DB.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM contest_registrations 
					WHERE user_id = $1 AND contest_id = $2
				)
			`, userID, *req.ContestID).Scan(&registered)
			
			if err != nil || !registered {
				c.JSON(http.StatusForbidden, gin.H{"error": "You must register for this contest to submit code."})
				return
			}
		}

		if status == "ENDED" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has ended. Practice submissions must be made outside the contest."})
			return
		}
	}

	// 2. Insert into Postgres as PENDING inside a Transaction
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var submissionID int
	query := `INSERT INTO submissions (user_id, problem_id, contest_id, code, language, status) 
			  VALUES ($1, $2, $3, $4, $5, 'PENDING') RETURNING id`
	err = tx.QueryRow(query, userID, req.ProblemID, req.ContestID, req.Code, req.Language).Scan(&submissionID)
	if err != nil {
		fmt.Printf("DB Error in SubmitCode: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// 3. Push to Redis Queue
	jobData, _ := json.Marshal(map[string]interface{}{
		"submission_id": submissionID,
		"code":          req.Code,
		"language":      req.Language,
		"problem_id":    req.ProblemID,
	})

	err = db.RedisClient.LPush(context.Background(), "submissions_queue", jobData).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue submission, please try again."})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit submission"})
		return
	}

	// 3. Return success with pending status
	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Submission queued successfully",
		"submission_id": submissionID,
		"status":        "PENDING",
	})
}

// RunRequest defines the payload for testing code against custom input
type RunRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Input    string `json:"input"`
}

// RunCode handles executing code against custom input without saving it to the database
func RunCode(c *gin.Context) {
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Acquire global execution capacity
	acqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := judge.AcquireExecutionToken(acqCtx)
	if err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Server is currently busy executing code. Please try again.",
		})
		return
	}
	defer judge.ReleaseExecutionToken()

	// Use our newly created Docker Sandbox
	// Note: This blocks the HTTP request until execution finishes.

	compRes, err := judge.CompileCode(req.Code, req.Language)
	if err != nil || compRes.Error != "" {
		errMsg := compRes.Error
		if err != nil {
			errMsg = fmt.Sprintf("Sandbox Error: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{
			"output": "",
			"stderr": errMsg,
		})
		return
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, req.Language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start sandbox session: %v", err)})
		return
	}
	defer session.Close()

	res, err := session.RunTestcase(req.Input)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"output": "",
			"stderr": fmt.Sprintf("Sandbox Error: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": res.Stdout,
		"stderr": res.Stderr,
	})
}

// GetSubmissionStatus handles polling for submission results
func GetSubmissionStatus(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetInt("user_id") // From auth middleware

	var status string
	var executionTimeMs *int
	var subUserID int

	query := `SELECT status, execution_time_ms, user_id FROM submissions WHERE id = $1`
	err := db.DB.QueryRow(query, id).Scan(&status, &executionTimeMs, &subUserID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	if subUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            status,
		"execution_time_ms": executionTimeMs,
	})
}

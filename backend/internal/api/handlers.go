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

	// 2. Insert into Postgres as PENDING
	var submissionID int
	query := `INSERT INTO submissions (user_id, problem_id, code, language, status) 
			  VALUES ($1, $2, $3, $4, 'PENDING') RETURNING id`
	err = db.DB.QueryRow(query, userID, req.ProblemID, req.Code, req.Language).Scan(&submissionID)
	if err != nil {
		fmt.Printf("DB Error in SubmitCode: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// 2. Push to Redis Queue
	jobData, _ := json.Marshal(map[string]interface{}{
		"submission_id": submissionID,
		"code":          req.Code,
		"language":      req.Language,
		"problem_id":    req.ProblemID,
	})

	err = db.RedisClient.LPush(context.Background(), "submissions_queue", jobData).Err()
	if err != nil {
		// Log error, but don't fail the request since it's already in DB
		// A background cron could sweep stuck 'PENDING' jobs later.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue submission"})
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

	// Use our newly created Docker Sandbox
	// Note: This blocks the HTTP request until execution finishes (up to 2.0s)
	// which is perfectly fine for a manual "Run" check.
	res, err := judge.RunSecurely(req.Code, req.Language, req.Input)

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
	var status string
	var executionTimeMs *int
	var byteCount *int

	query := `SELECT status, execution_time_ms, byte_count FROM submissions WHERE id = $1`
	err := db.DB.QueryRow(query, id).Scan(&status, &executionTimeMs, &byteCount)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            status,
		"execution_time_ms": executionTimeMs,
		"byte_count":        byteCount,
	})
}

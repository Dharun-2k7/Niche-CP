package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

// HealthCheck responds with a simple OK
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "API is healthy!"})
}

// SubmitRequest defines the JSON payload for a code submission
type SubmitRequest struct {
	UserID    int    `json:"user_id" binding:"required"`
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

	// 1. Insert into Postgres as PENDING
	var submissionID int
	query := `INSERT INTO submissions (user_id, problem_id, code, language, status) 
			  VALUES ($1, $2, $3, $4, 'PENDING') RETURNING id`
	err := db.DB.QueryRow(query, req.UserID, req.ProblemID, req.Code, req.Language).Scan(&submissionID)
	if err != nil {
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

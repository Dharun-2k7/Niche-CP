package api

import (
	"net/http"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

func GetUpcomingContests(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, title, type, start_time, duration_minutes, status 
		FROM contests 
		WHERE status IN ('CREATED', 'UPCOMING', 'RUNNING')
		ORDER BY start_time ASC 
		LIMIT 5
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch upcoming contests"})
		return
	}
	defer rows.Close()

	contests := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, duration int
		var title, typeStr, status string
		var startTime time.Time
		if err := rows.Scan(&id, &title, &typeStr, &startTime, &duration, &status); err == nil {
			contests = append(contests, map[string]interface{}{
				"id":               id,
				"title":            title,
				"type":             typeStr,
				"start_time":       startTime.Format(time.RFC3339),
				"duration_minutes": duration,
				"status":           status,
			})
		}
	}
	c.JSON(http.StatusOK, contests)
}

func GetRecentProblems(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, title, difficulty, tags, created_at 
		FROM problems 
		ORDER BY created_at DESC 
		LIMIT 5
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent problems"})
		return
	}
	defer rows.Close()

	problems := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var title, difficulty, tags string
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &difficulty, &tags, &createdAt); err == nil {
			problems = append(problems, map[string]interface{}{
				"id":         id,
				"title":      title,
				"difficulty": difficulty,
				"tags":       tags,
				"created_at": createdAt.Format(time.RFC3339),
			})
		}
	}
	c.JSON(http.StatusOK, problems)
}

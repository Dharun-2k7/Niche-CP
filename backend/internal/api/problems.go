package api

import (
	"database/sql"
	"net/http"
	"time"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

func GetProblem(c *gin.Context) {
	id := c.Param("id")

	var title, description, difficulty, inputFormat, outputFormat, constraints, sampleTestcases string
	var tags string

	err := db.DB.QueryRow(`
		SELECT title, difficulty, tags, description, input_format, output_format, constraints, sample_testcases 
		FROM problems WHERE id = $1
	`, id).Scan(&title, &difficulty, &tags, &description, &inputFormat, &outputFormat, &constraints, &sampleTestcases)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               id,
		"title":            title,
		"difficulty":       difficulty,
		"tags":             tags,
		"description":      description,
		"input_format":     inputFormat,
		"output_format":    outputFormat,
		"constraints":      constraints,
		"sample_testcases": sampleTestcases,
	})
}

func GetAllProblems(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, title, difficulty, tags FROM problems ORDER BY id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
		return
	}
	defer rows.Close()

	var problems []map[string]interface{}
	for rows.Next() {
		var id int
		var title, difficulty, tags string
		if err := rows.Scan(&id, &title, &difficulty, &tags); err == nil {
			problems = append(problems, map[string]interface{}{
				"id":         id,
				"title":      title,
				"difficulty": difficulty,
				"tags":       tags,
			})
		}
	}
	c.JSON(http.StatusOK, problems)
}

func GetAllContests(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, title, type, start_time, duration_minutes FROM contests ORDER BY start_time DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contests"})
		return
	}
	defer rows.Close()

	var contests []map[string]interface{}
	for rows.Next() {
		var id, duration int
		var title, typeStr, startTime string
		if err := rows.Scan(&id, &title, &typeStr, &startTime, &duration); err == nil {
			contests = append(contests, map[string]interface{}{
				"id":               id,
				"title":            title,
				"type":             typeStr,
				"start_time":       startTime,
				"duration_minutes": duration,
			})
		}
	}
	c.JSON(http.StatusOK, contests)
}

type RegisterContestRequest struct {
	ContestID int `json:"contest_id" binding:"required"`
}

func RegisterForContest(c *gin.Context) {
	var req RegisterContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	var startTime, regOpenTime time.Time
	err := db.DB.QueryRow(`SELECT start_time, registration_open_time FROM contests WHERE id = $1`, req.ContestID).Scan(&startTime, &regOpenTime)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	now := time.Now()
	if now.Before(regOpenTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration has not opened yet"})
		return
	}
	if now.After(startTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration is closed (contest has started)"})
		return
	}

	// Assuming a contest_registrations table. Let's create it if missing or just mock the logic.
	// We'll create contest_registrations in schema.sql next.
	_, err = db.DB.Exec(`
		INSERT INTO contest_registrations (user_id, contest_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, contest_id) DO NOTHING
	`, userID, req.ContestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully registered for contest"})
}

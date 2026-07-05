package api

import (
	"database/sql"
	"net/http"
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

package api

import (
	"database/sql"
	"net/http"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

func GetProblem(c *gin.Context) {
	id := c.Param("id")

	var title, description, testCases string
	var contestID *int
	err := db.DB.QueryRow(`
		SELECT title, description, test_cases, contest_id 
		FROM problems WHERE id = $1
	`, id).Scan(&title, &description, &testCases, &contestID)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"title":       title,
		"description": description,
		"test_cases":  testCases,
		"contest_id":  contestID,
	})
}

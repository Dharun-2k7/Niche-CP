package api

import (
	"net/http"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

func GetAllUsers(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, email, name, roll_no, batch, is_college_verified FROM users`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	type UserData struct {
		ID                int    `json:"id"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		RollNo            string `json:"roll_no"`
		Batch             string `json:"batch"`
		IsCollegeVerified bool   `json:"is_college_verified"`
	}

	var users []UserData
	for rows.Next() {
		var u UserData
		var rollNo, batch *string
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &rollNo, &batch, &u.IsCollegeVerified); err != nil {
			continue
		}
		if rollNo != nil {
			u.RollNo = *rollNo
		}
		if batch != nil {
			u.Batch = *batch
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
}

type CreateProblemRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	TestCases   string `json:"test_cases" binding:"required"` // Expecting JSON string
	ContestID   *int   `json:"contest_id"`
}

func CreateProblem(c *gin.Context) {
	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert into DB
	var problemID int
	err := db.DB.QueryRow(`
		INSERT INTO problems (title, description, test_cases, contest_id)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id
	`, req.Title, req.Description, req.TestCases, req.ContestID).Scan(&problemID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem created successfully!", "problem_id": problemID})
}

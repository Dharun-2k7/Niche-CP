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
	Title           string `json:"title" binding:"required"`
	Difficulty      string `json:"difficulty"`
	Tags            string `json:"tags"`             // JSON array string
	Description     string `json:"description" binding:"required"`
	InputFormat     string `json:"input_format"`
	OutputFormat    string `json:"output_format"`
	Constraints     string `json:"constraints"`
	SampleTestcases string `json:"sample_testcases"` // JSON array string
	HiddenTestcases string `json:"hidden_testcases" binding:"required"` // JSON array string
	ContestID       *int   `json:"contest_id"`
}

func CreateProblem(c *gin.Context) {
	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Tags == "" { req.Tags = "[]" }
	if req.SampleTestcases == "" { req.SampleTestcases = "[]" }
	if req.Difficulty == "" { req.Difficulty = "Medium" }

	// Insert into DB
	var problemID int
	err := db.DB.QueryRow(`
		INSERT INTO problems (title, difficulty, tags, description, input_format, output_format, constraints, sample_testcases, hidden_testcases)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7, $8::jsonb, $9::jsonb)
		RETURNING id
	`, req.Title, req.Difficulty, req.Tags, req.Description, req.InputFormat, req.OutputFormat, req.Constraints, req.SampleTestcases, req.HiddenTestcases).Scan(&problemID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	// Link to contest if provided
	if req.ContestID != nil {
		var orderIndex int
		db.DB.QueryRow(`SELECT COALESCE(MAX(order_index), 0) + 1 FROM contest_problems WHERE contest_id = $1`, *req.ContestID).Scan(&orderIndex)
		db.DB.Exec(`INSERT INTO contest_problems (contest_id, problem_id, order_index) VALUES ($1, $2, $3)`, *req.ContestID, problemID, orderIndex)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem created successfully!", "problem_id": problemID})
}

type CreateContestRequest struct {
	Title           string `json:"title" binding:"required"`
	Type            string `json:"type" binding:"required"`
	StartTime       string `json:"start_time" binding:"required"`
	DurationMinutes int    `json:"duration_minutes" binding:"required"`
}

func CreateContest(c *gin.Context) {
	var req CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var contestID int
	err := db.DB.QueryRow(`
		INSERT INTO contests (title, type, start_time, duration_minutes)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.Title, req.Type, req.StartTime, req.DurationMinutes).Scan(&contestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest created successfully!", "contest_id": contestID})
}

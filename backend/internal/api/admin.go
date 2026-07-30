package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

func GetAllUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "50")
	
	page := 1
	limit := 50
	fmt.Sscanf(pageStr, "%d", &page)
	fmt.Sscanf(limitStr, "%d", &limit)
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 50 }
	offset := (page - 1) * limit

	rows, err := db.DB.Query(`SELECT id, email, name, roll_no, batch, is_college_verified FROM users ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
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
	StartTime            string `json:"start_time" binding:"required"`
	DurationMinutes      int    `json:"duration_minutes" binding:"required"`
	RegistrationOpenTime string `json:"registration_open_time"`
	Description          string `json:"description"`
	Status               string `json:"status"`
}

func CreateContest(c *gin.Context) {
	var req CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	importTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format, must be RFC3339"})
		return
	}

	var regOpenTime time.Time
	if req.RegistrationOpenTime == "" {
		regOpenTime = importTime.Add(-48 * time.Hour)
	} else {
		regOpenTime, err = time.Parse(time.RFC3339, req.RegistrationOpenTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration_open_time format, must be RFC3339"})
			return
		}
	}

	status := req.Status
	if status == "" {
		status = "CREATED"
	}

	endTime := importTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	var contestID int
	err = db.DB.QueryRow(`
		INSERT INTO contests (title, type, start_time, end_time, duration_minutes, registration_open_time, description, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, req.Title, req.Type, importTime.UTC(), endTime.UTC(), req.DurationMinutes, regOpenTime.UTC(), req.Description, status).Scan(&contestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest created successfully!", "contest_id": contestID})
}

type UpdateContestRequest struct {
	Title                string `json:"title" binding:"required"`
	Type                 string `json:"type" binding:"required"`
	StartTime            string `json:"start_time" binding:"required"`
	DurationMinutes      int    `json:"duration_minutes" binding:"required"`
	RegistrationOpenTime string `json:"registration_open_time"`
	Description          string `json:"description"`
	Status               string `json:"status"`
}

func UpdateContest(c *gin.Context) {
	contestID := c.Param("id")
	var req UpdateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	importTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format, must be RFC3339"})
		return
	}

	var regOpenTime time.Time
	if req.RegistrationOpenTime == "" {
		regOpenTime = importTime.Add(-48 * time.Hour)
	} else {
		regOpenTime, err = time.Parse(time.RFC3339, req.RegistrationOpenTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid registration_open_time format, must be RFC3339"})
			return
		}
	}

	status := req.Status
	if status == "" {
		status = "CREATED"
	}

	endTime := importTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	res, err := db.DB.Exec(`
		UPDATE contests 
		SET title = $1, type = $2, start_time = $3, end_time = $4, duration_minutes = $5, registration_open_time = $6, description = $7, status = $8
		WHERE id = $9
	`, req.Title, req.Type, importTime.UTC(), endTime.UTC(), req.DurationMinutes, regOpenTime.UTC(), req.Description, status, contestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contest"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest updated successfully"})
}

type UpdatePermissionsRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Permissions []string `json:"permissions" binding:"required"`
}

func UpdateUserPermissions(c *gin.Context) {
	var req UpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	superAdmin := os.Getenv("SUPER_ADMIN_EMAIL")
	if superAdmin == "" {
		superAdmin = "dharunkaarthick07@gmail.com"
	}
	if req.Email == superAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify permissions of the super admin."})
		return
	}

	// Validate permissions
	validPermissions := map[string]bool{
		"create_problem": true,
		"create_contest": true,
		"manage_users":   true,
		"manage_admins":  true,
	}

	for _, p := range req.Permissions {
		if !validPermissions[p] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission: " + p})
			return
		}
	}

	// Convert to JSON
	permJSON, _ := json.Marshal(req.Permissions)

	res, err := db.DB.Exec(`UPDATE users SET permissions = $1::jsonb WHERE email = $2`, string(permJSON), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permissions"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated successfully"})
}

type PromoteAdminRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func PromoteToAdmin(c *gin.Context) {
	var req PromoteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerEmail := c.GetString("email")
	superAdmin := os.Getenv("SUPER_ADMIN_EMAIL")
	if superAdmin == "" {
		superAdmin = "dharunkaarthick07@gmail.com"
	}
	
	// Only super admin can promote
	if callerEmail != superAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the Super Admin can promote users to admin."})
		return
	}

	res, err := db.DB.Exec(`UPDATE users SET role = 'admin' WHERE email = $1`, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User successfully promoted to Admin"})
}

// GetContestDetails returns contest metadata for the delete confirmation modal
func GetContestDetails(c *gin.Context) {
	contestID := c.Param("id")

	var title string
	err := db.DB.QueryRow(`SELECT title FROM contests WHERE id = $1`, contestID).Scan(&title)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	var problemCount int
	db.DB.QueryRow(`SELECT COUNT(*) FROM contest_problems WHERE contest_id = $1`, contestID).Scan(&problemCount)

	var submissionCount int
	db.DB.QueryRow(`SELECT COUNT(*) FROM submissions WHERE contest_id = $1`, contestID).Scan(&submissionCount)

	// Get attached problems
	rows, err := db.DB.Query(`
		SELECT p.id, p.title
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		WHERE cp.contest_id = $1
		ORDER BY cp.order_index ASC
	`, contestID)

	type ProblemInfo struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	problems := make([]ProblemInfo, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p ProblemInfo
			if err := rows.Scan(&p.ID, &p.Title); err == nil {
				problems = append(problems, p)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"title":            title,
		"problem_count":    problemCount,
		"submission_count": submissionCount,
		"problems":         problems,
	})
}

type DeleteContestRequest struct {
	DeleteMode string `json:"delete_mode"` // "contest_only", "selected_problems", "all_problems"
	ProblemIDs []int  `json:"problem_ids"` // Only used for "selected_problems"
}

// DeleteContest handles contest deletion with three modes using database transactions
func DeleteContest(c *gin.Context) {
	contestID := c.Param("id")

	var req DeleteContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.DeleteMode = "contest_only"
	}
	if req.DeleteMode == "" {
		req.DeleteMode = "contest_only"
	}

	// Verify contest exists
	var exists bool
	err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM contests WHERE id = $1)`, contestID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	switch req.DeleteMode {
	case "contest_only":
		if _, err := tx.Exec(`DELETE FROM contest_problems WHERE contest_id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest-problem mappings"})
			return
		}
		if _, err := tx.Exec(`DELETE FROM contests WHERE id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest"})
			return
		}

	case "selected_problems":
		if len(req.ProblemIDs) == 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No problem IDs provided for selected_problems mode"})
			return
		}
		// Validate that selected problems belong to this contest
		for _, pid := range req.ProblemIDs {
			var belongs bool
			err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM contest_problems WHERE contest_id = $1 AND problem_id = $2)`, contestID, pid).Scan(&belongs)
			if err != nil || !belongs {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Problem ID %d does not belong to this contest", pid)})
				return
			}
		}
		// Delete test cases and problems for selected IDs
		for _, pid := range req.ProblemIDs {
			if _, err := tx.Exec(`DELETE FROM contest_problems WHERE contest_id = $1 AND problem_id = $2`, contestID, pid); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink problem"})
				return
			}
			if _, err := tx.Exec(`DELETE FROM problems WHERE id = $1`, pid); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problem"})
				return
			}
		}
		// Delete remaining contest mappings and the contest itself
		if _, err := tx.Exec(`DELETE FROM contest_problems WHERE contest_id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up mappings"})
			return
		}
		if _, err := tx.Exec(`DELETE FROM contests WHERE id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest"})
			return
		}

	case "all_problems":
		// Get all problem IDs for this contest
		rows, err := tx.Query(`SELECT problem_id FROM contest_problems WHERE contest_id = $1`, contestID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contest problems"})
			return
		}
		var problemIDs []int
		for rows.Next() {
			var pid int
			if err := rows.Scan(&pid); err == nil {
				problemIDs = append(problemIDs, pid)
			}
		}
		rows.Close()

		// Delete all linked problems
		for _, pid := range problemIDs {
			if _, err := tx.Exec(`DELETE FROM problems WHERE id = $1`, pid); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problem"})
				return
			}
		}
		// Delete mappings and contest
		if _, err := tx.Exec(`DELETE FROM contest_problems WHERE contest_id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mappings"})
			return
		}
		if _, err := tx.Exec(`DELETE FROM contests WHERE id = $1`, contestID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest"})
			return
		}

	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delete_mode. Use: contest_only, selected_problems, or all_problems"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest deleted successfully"})
}

func GetAdminProblem(c *gin.Context) {
	id := c.Param("id")

	var title, description, difficulty, inputFormat, outputFormat, constraints, sampleTestcases, hiddenTestcases string
	var tags string

	err := db.DB.QueryRow(`
		SELECT title, difficulty, tags, description, input_format, output_format, constraints, sample_testcases, hidden_testcases 
		FROM problems WHERE id = $1
	`, id).Scan(&title, &difficulty, &tags, &description, &inputFormat, &outputFormat, &constraints, &sampleTestcases, &hiddenTestcases)

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
		"hidden_testcases": hiddenTestcases,
	})
}

type UpdateProblemRequest struct {
	Title           string      `json:"title" binding:"required"`
	Difficulty      string      `json:"difficulty"`
	Tags            string      `json:"tags"` // JSON string array
	Description     string      `json:"description" binding:"required"`
	InputFormat     string      `json:"input_format"`
	OutputFormat    string      `json:"output_format"`
	Constraints     string      `json:"constraints"`
	SampleTestcases string      `json:"sample_testcases"` // JSON string array of objects
	HiddenTestcases string      `json:"hidden_testcases"` // JSON string array of objects
	ContestID       *int        `json:"contest_id"`
}

func UpdateProblem(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	res, err := tx.Exec(`
		UPDATE problems 
		SET title=$1, difficulty=$2, tags=$3, description=$4, input_format=$5, output_format=$6, constraints=$7, sample_testcases=$8, hidden_testcases=$9
		WHERE id=$10
	`, req.Title, req.Difficulty, req.Tags, req.Description, req.InputFormat, req.OutputFormat, req.Constraints, req.SampleTestcases, req.HiddenTestcases, id)
	
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update problem"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Update contest mapping if contest_id is provided
	if req.ContestID != nil {
		var existingCount int
		tx.QueryRow(`SELECT COUNT(*) FROM contest_problems WHERE problem_id = $1 AND contest_id = $2`, id, *req.ContestID).Scan(&existingCount)
		
		if existingCount == 0 {
			var maxOrder sql.NullInt64
			tx.QueryRow(`SELECT MAX(order_index) FROM contest_problems WHERE contest_id = $1`, *req.ContestID).Scan(&maxOrder)
			
			nextOrder := 1
			if maxOrder.Valid {
				nextOrder = int(maxOrder.Int64) + 1
			}

			_, err = tx.Exec(`
				INSERT INTO contest_problems (contest_id, problem_id, order_index)
				VALUES ($1, $2, $3)
			`, *req.ContestID, id, nextOrder)
			
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link problem to contest"})
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem updated successfully!"})
}

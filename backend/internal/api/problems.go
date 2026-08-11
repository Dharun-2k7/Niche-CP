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

	problems := make([]map[string]interface{}, 0)
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

func GetContestProblems(c *gin.Context) {
	contestID := c.Param("id")

	rows, err := db.DB.Query(`
		SELECT p.id, p.title, p.difficulty, p.tags
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		WHERE cp.contest_id = $1
		ORDER BY cp.order_index ASC
	`, contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contest problems"})
		return
	}
	defer rows.Close()

	problems := make([]map[string]interface{}, 0)
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

// GetContestLeaderboard returns ranked users for a contest based on accepted submissions
func GetContestLeaderboard(c *gin.Context) {
	contestID := c.Param("id")

	rows, err := db.DB.Query(`
		SELECT u.name, 
			COUNT(DISTINCT s.problem_id) AS solved_count,
			SUM(CASE WHEN s.execution_time_ms IS NOT NULL THEN s.execution_time_ms ELSE 0 END) AS total_time
		FROM submissions s
		JOIN users u ON s.user_id = u.id
		WHERE s.contest_id = $1 AND s.status = 'ACCEPTED'
		GROUP BY u.id, u.name
		ORDER BY solved_count DESC, total_time ASC
	`, contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}
	defer rows.Close()

	type LeaderboardEntry struct {
		Rank      int    `json:"rank"`
		Name      string `json:"name"`
		Solved    int    `json:"solved"`
		TotalTime int    `json:"total_time"`
	}

	entries := make([]LeaderboardEntry, 0)
	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Name, &entry.Solved, &entry.TotalTime); err == nil {
			entry.Rank = rank
			entries = append(entries, entry)
			rank++
		}
	}
	c.JSON(http.StatusOK, entries)
}

// GetContestSubmissions returns recent submissions for the contest (live commentary feed)
func GetContestSubmissions(c *gin.Context) {
	contestID := c.Param("id")

	rows, err := db.DB.Query(`
		SELECT u.name, p.title, s.status, s.created_at
		FROM submissions s
		JOIN users u ON s.user_id = u.id
		JOIN problems p ON s.problem_id = p.id
		WHERE s.contest_id = $1
		ORDER BY s.created_at DESC
		LIMIT 50
	`, contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contest submissions"})
		return
	}
	defer rows.Close()

	type SubmissionEvent struct {
		Username  string    `json:"username"`
		Problem   string    `json:"problem"`
		Verdict   string    `json:"verdict"`
		Timestamp time.Time `json:"timestamp"`
	}

	events := make([]SubmissionEvent, 0)
	for rows.Next() {
		var ev SubmissionEvent
		if err := rows.Scan(&ev.Username, &ev.Problem, &ev.Verdict, &ev.Timestamp); err == nil {
			events = append(events, ev)
		}
	}
	c.JSON(http.StatusOK, events)
}

// GetMyContestSubmissions returns the logged-in user's submissions for a specific contest
func GetMyContestSubmissions(c *gin.Context) {
	contestID := c.Param("id")
	userID := c.GetInt("user_id")

	rows, err := db.DB.Query(`
		SELECT p.title, s.status, s.language, s.execution_time_ms, s.created_at
		FROM submissions s
		JOIN problems p ON s.problem_id = p.id
		WHERE s.contest_id = $1 AND s.user_id = $2
		ORDER BY s.created_at DESC
		LIMIT 50
	`, contestID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your submissions"})
		return
	}
	defer rows.Close()

	type MySubmission struct {
		Problem       string    `json:"problem"`
		Verdict       string    `json:"verdict"`
		Language      string    `json:"language"`
		ExecutionTime *int      `json:"execution_time_ms"`
		Timestamp     time.Time `json:"timestamp"`
	}

	subs := make([]MySubmission, 0)
	for rows.Next() {
		var s MySubmission
		if err := rows.Scan(&s.Problem, &s.Verdict, &s.Language, &s.ExecutionTime, &s.Timestamp); err == nil {
			subs = append(subs, s)
		}
	}
	c.JSON(http.StatusOK, subs)
}

func GetAllContests(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, title, type, description, start_time, end_time, duration_minutes, status, registration_open_time FROM contests ORDER BY start_time DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contests"})
		return
	}
	defer rows.Close()

	now := time.Now().UTC()
	contests := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, duration int
		var title, typeStr, status string
		var description sql.NullString
		var startTime, endTime time.Time
		var regOpenTime sql.NullTime
		if err := rows.Scan(&id, &title, &typeStr, &description, &startTime, &endTime, &duration, &status, &regOpenTime); err == nil {
			desc := ""
			if description.Valid {
				desc = description.String
			}
			computedStatus := status
			if now.Before(startTime) {
				if status != "CREATED" {
					computedStatus = "UPCOMING"
				}
			} else if !now.Before(startTime) && now.Before(endTime) {
				computedStatus = "RUNNING"
			} else if !now.Before(endTime) {
				computedStatus = "ENDED"
			}

			regOpen := false
			if regOpenTime.Valid {
				regOpen = !now.Before(regOpenTime.Time) && now.Before(startTime) && (computedStatus == "UPCOMING" || computedStatus == "CREATED")
			} else {
				regOpen = now.Before(startTime) && (computedStatus == "UPCOMING" || computedStatus == "CREATED")
			}

			contests = append(contests, map[string]interface{}{
				"id":                id,
				"title":             title,
				"type":              typeStr,
				"description":       desc,
				"start_time":        startTime.Format(time.RFC3339),
				"end_time":          endTime.Format(time.RFC3339),
				"duration_minutes":  duration,
				"status":            computedStatus,
				"registration_open": regOpen,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"server_time": time.Now().UTC().Format(time.RFC3339),
		"contests":    contests,
	})
}

func GetMyRegistrations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`SELECT contest_id FROM contest_registrations WHERE user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch registrations"})
		return
	}
	defer rows.Close()

	var contestIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			contestIDs = append(contestIDs, id)
		}
	}
	// Return empty array instead of null
	if contestIDs == nil {
		contestIDs = make([]int, 0)
	}

	c.JSON(http.StatusOK, gin.H{"registered_contests": contestIDs})
}

func RegisterForContest(c *gin.Context) {
	contestID := c.Param("id")
	userID, _ := c.Get("user_id")

	var startTime, regOpenTime time.Time
	var status string
	err := db.DB.QueryRow(`SELECT start_time, registration_open_time, status FROM contests WHERE id = $1`, contestID).Scan(&startTime, &regOpenTime, &status)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	now := time.Now().UTC()
	if now.Before(regOpenTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration has not opened yet"})
		return
	}
	if status == "RUNNING" || status == "ENDED" || now.After(startTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration is closed (contest is not UPCOMING)"})
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO contest_registrations (user_id, contest_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, contest_id) DO NOTHING
	`, userID, contestID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully registered for contest"})
}

func CheckContestAccess(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
}

type ViolationRequest struct {
	EventType string `json:"event_type" binding:"required"`
}

func LogContestViolation(c *gin.Context) {
	userID := c.GetInt("user_id")
	contestID := c.Param("id")

	var req ViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	_, err := db.DB.Exec(`
		INSERT INTO contest_violations (user_id, contest_id, event_type)
		VALUES ($1, $2, $3)
	`, userID, contestID, req.EventType)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log violation"})
		return
	}

	// Check if this user has exceeded the 3 warning limit
	var count int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM contest_violations WHERE user_id = $1 AND contest_id = $2
	`, userID, contestID).Scan(&count)
	
	if err == nil && count >= 3 {
		// Log disqualified or ban logic
		// We could update a contest_registrations table to mark them disqualified
		c.JSON(http.StatusOK, gin.H{"message": "Logged successfully", "disqualified": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged successfully", "disqualified": false})
}

func GetContestViolations(c *gin.Context) {
	contestID := c.Param("id")

	rows, err := db.DB.Query(`
		SELECT v.id, v.user_id, u.email, v.event_type, v.timestamp
		FROM contest_violations v
		JOIN users u ON v.user_id = u.id
		WHERE v.contest_id = $1
		ORDER BY v.timestamp DESC
	`, contestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch violations"})
		return
	}
	defer rows.Close()

	var violations []map[string]interface{}
	for rows.Next() {
		var id, userID int
		var email, eventType string
		var timestamp time.Time
		if err := rows.Scan(&id, &userID, &email, &eventType, &timestamp); err != nil {
			continue
		}
		violations = append(violations, map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"email":      email,
			"event_type": eventType,
			"timestamp":  timestamp,
		})
	}

	if violations == nil {
		violations = make([]map[string]interface{}, 0)
	}

	c.JSON(http.StatusOK, violations)
}

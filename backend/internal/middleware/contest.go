package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

// RequireContestLifecycle strictly enforces contest access rules based on status and user registration.
func RequireContestLifecycle() gin.HandlerFunc {
	return func(c *gin.Context) {
		contestID := c.Param("id")
		if contestID == "" {
			c.Next()
			return
		}

		// Optional Auth Parsing
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				userID, _, err := auth.ValidateToken(parts[1])
				if err == nil {
					c.Set("user_id", userID)
				}
			}
		}

		var status string
		err := db.DB.QueryRow("SELECT status FROM contests WHERE id = $1", contestID).Scan(&status)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking contest"})
			}
			c.Abort()
			return
		}

		if status == "CREATED" || status == "UPCOMING" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet."})
			c.Abort()
			return
		}

		if status == "RUNNING" {
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "You must log in and register to enter the contest arena."})
				c.Abort()
				return
			}

			var registered bool
			err = db.DB.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM contest_registrations 
					WHERE user_id = $1 AND contest_id = $2
				)
			`, userID, contestID).Scan(&registered)
			
			if err != nil || !registered {
				c.JSON(http.StatusForbidden, gin.H{"error": "You must register for this contest to view problems or submit code."})
				c.Abort()
				return
			}
		}

		// If status == "ENDED", access is allowed for practice
		c.Next()
	}
}

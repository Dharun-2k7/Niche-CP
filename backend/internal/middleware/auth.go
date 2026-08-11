package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

// RequireAuth ensures that a valid JWT token is present in the Authorization header
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			c.Abort()
			return
		}

		userID, email, err := auth.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set the user info into the context for downstream handlers
		c.Set("user_id", userID)
		c.Set("email", email)

		c.Next()
	}
}

// RequireAdmin ensures that the authenticated user is an admin
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure they are authenticated
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			c.Abort()
			return
		}

		userID, email, err := auth.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		var role string
		err = db.DB.QueryRow(`SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		normRole := strings.ToLower(strings.TrimSpace(role))
		superadminEmailsEnv := os.Getenv("SUPERADMIN_EMAILS")
		isSuperAdminEmail := false
		if superadminEmailsEnv != "" {
			for _, e := range strings.Split(superadminEmailsEnv, ",") {
				if strings.EqualFold(strings.TrimSpace(e), email) {
					isSuperAdminEmail = true
					break
				}
			}
		}

		isAdmin := (normRole == "admin" || normRole == "superadmin" || isSuperAdminEmail)

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		if isSuperAdminEmail && normRole != "superadmin" {
			_, _ = db.DB.Exec(`UPDATE users SET role = 'superadmin' WHERE id = $1`, userID)
			role = "superadmin"
		}

		c.Set("user_id", userID)
		c.Set("email", email)
		c.Set("is_admin", true)
		c.Set("role", role)

		c.Next()
	}
}

// RequireSuperAdmin ensures that the authenticated user is a superadmin
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			c.Abort()
			return
		}

		userID, email, err := auth.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		var role string
		err = db.DB.QueryRow(`SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		normRole := strings.ToLower(strings.TrimSpace(role))
		superadminEmailsEnv := os.Getenv("SUPERADMIN_EMAILS")
		isSuperAdminEmail := false
		if superadminEmailsEnv != "" {
			for _, e := range strings.Split(superadminEmailsEnv, ",") {
				if strings.EqualFold(strings.TrimSpace(e), email) {
					isSuperAdminEmail = true
					break
				}
			}
		}

		isSuperAdmin := (normRole == "superadmin" || isSuperAdminEmail)

		if !isSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Super admin access required"})
			c.Abort()
			return
		}

		if isSuperAdminEmail && normRole != "superadmin" {
			_, _ = db.DB.Exec(`UPDATE users SET role = 'superadmin' WHERE id = $1`, userID)
			role = "superadmin"
		}

		c.Set("user_id", userID)
		c.Set("email", email)
		c.Set("is_admin", true)
		c.Set("role", role)

		c.Next()
	}
}

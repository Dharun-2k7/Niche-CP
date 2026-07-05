package middleware

import (
	"net/http"
	"strings"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
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

		// Hardcoded super admin
		if email != "dharunkaarthick07@gmail.com" {
			// Check if they have admin permissions in DB?
			// For now, only allow the hardcoded email
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("email", email)
		c.Set("is_admin", true)

		c.Next()
	}
}

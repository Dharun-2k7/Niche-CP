package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/mailer"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Generate a 6-digit OTP
func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// Generate a secure random token
func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type OTPSendRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SendOTP generates an OTP, stores it in Redis, and emails it
func SendOTP(c *gin.Context) {
	var req OTPSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists
	var userID int
	err := db.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, req.Email).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No account found with this email."})
		return
	}

	otp := generateOTP()
	key := fmt.Sprintf("otp:%s", req.Email)

	// Store in Redis with 5-minute TTL
	err = db.RedisClient.Set(context.Background(), key, otp, 5*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Send Email
	body := fmt.Sprintf("Your Login OTP is: %s\n\nThis code will expire in 5 minutes.", otp)
	go mailer.SendEmail(req.Email, "Your NicheCP Login Code", body)

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully to your email."})
}

type OTPVerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

// VerifyOTP checks the OTP against Redis and logs the user in
func VerifyOTP(c *gin.Context) {
	var req OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := fmt.Sprintf("otp:%s", req.Email)
	storedOTP, err := db.RedisClient.Get(context.Background(), key).Result()
	if err != nil || storedOTP != req.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP."})
		return
	}

	// Clear OTP after successful use
	db.RedisClient.Del(context.Background(), key)

	// Get User ID
	var userID int
	err = db.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, req.Email).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User lookup failed"})
		return
	}

	jwtToken, err := auth.GenerateToken(userID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken, "message": "Login successful!"})
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword generates a reset link and emails it
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure user exists
	var userID int
	err := db.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, req.Email).Scan(&userID)
	if err != nil {
		// Return 200 to prevent email enumeration attacks, but we log it internally
		c.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
		return
	}

	token := generateSecureToken()
	key := fmt.Sprintf("reset:%s", token)

	// Store token in Redis with 15 min TTL, mapping to their email
	err = db.RedisClient.Set(context.Background(), key, req.Email, 15*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Send Email (link points to local frontend for now)
	resetLink := fmt.Sprintf("http://localhost:3000/reset.html?token=%s", token)
	body := fmt.Sprintf("Click the link below to reset your password:\n\n%s\n\nThis link expires in 15 minutes.", resetLink)
	go mailer.SendEmail(req.Email, "Reset Your NicheCP Password", body)

	c.JSON(http.StatusOK, gin.H{"message": "If an account exists, a reset link has been sent."})
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPassword consumes the token and updates the Postgres password hash
func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := fmt.Sprintf("reset:%s", req.Token)
	email, err := db.RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset link."})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password"})
		return
	}

	// Update DB
	_, err = db.DB.Exec(`UPDATE users SET password_hash = $1 WHERE email = $2`, string(hashedPassword), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Delete token so it can't be reused
	db.RedisClient.Del(context.Background(), key)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. You can now login."})
}

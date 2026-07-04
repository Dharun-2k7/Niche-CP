package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/mailer"
	"github.com/gin-gonic/gin"
)

type UserProfile struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	Role              string `json:"role"`
	RollNo            string `json:"roll_no"`
	Batch             string `json:"batch"`
	CollegeEmail      string `json:"college_email"`
	IsCollegeVerified bool   `json:"is_college_verified"`
	TotalSolved       int    `json:"total_solved"`
}

func GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var profile UserProfile
	var rollNo, batch, collegeEmail *string

	err := db.DB.QueryRow(`
		SELECT name, email, role, roll_no, batch, college_email, is_college_verified 
		FROM users WHERE id = $1
	`, userID).Scan(&profile.Name, &profile.Email, &profile.Role, &rollNo, &batch, &collegeEmail, &profile.IsCollegeVerified)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile"})
		return
	}

	if rollNo != nil {
		profile.RollNo = *rollNo
	}
	if batch != nil {
		profile.Batch = *batch
	}
	if collegeEmail != nil {
		profile.CollegeEmail = *collegeEmail
	}

	// Mock total solved for now
	profile.TotalSolved = 15

	c.JSON(http.StatusOK, profile)
}

type VerifyCollegeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func VerifyCollegeEmail(c *gin.Context) {
	var req VerifyCollegeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// 1. Generate OTP
	otp := generateOTP()
	key := fmt.Sprintf("verify_college:%d", userID)

	// 2. Store in Redis mapping to the requested email
	err := db.RedisClient.Set(context.Background(), key, req.Email+":"+otp, 10*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// 3. Send Email
	body := fmt.Sprintf("Hello,\n\nPlease verify your NicheCP College Email connection using this OTP: %s\n\nThis code expires in 10 minutes.", otp)
	go mailer.SendEmail(req.Email, "NicheCP College Verification", body)

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent to your college email!"})
}

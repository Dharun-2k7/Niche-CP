package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	CFHandle          string `json:"cf_handle"`
	CFVerifyString    string `json:"cf_verify_string"`
	IsCFVerified      bool   `json:"is_cf_verified"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

func GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var profile UserProfile
	var rollNo, batch, collegeEmail, cfHandle, cfVerifyString, profilePic *string
	var isCFVerified *bool

	err := db.DB.QueryRow(`
		SELECT name, email, role, roll_no, batch, college_email, is_college_verified, cf_handle, cf_verify_string, is_cf_verified, profile_picture_url
		FROM users WHERE id = $1
	`, userID).Scan(&profile.Name, &profile.Email, &profile.Role, &rollNo, &batch, &collegeEmail, &profile.IsCollegeVerified, &cfHandle, &cfVerifyString, &isCFVerified, &profilePic)

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
	if cfHandle != nil {
		profile.CFHandle = *cfHandle
	}
	if cfVerifyString != nil {
		profile.CFVerifyString = *cfVerifyString
	}
	if isCFVerified != nil {
		profile.IsCFVerified = *isCFVerified
	}
	if profilePic != nil {
		profile.ProfilePictureURL = *profilePic
	}

	// Fetch total solved
	var totalSolved int
	db.DB.QueryRow(`SELECT COUNT(*) FROM user_problem_status WHERE user_id = $1 AND status = 'ACCEPTED'`, userID).Scan(&totalSolved)
	profile.TotalSolved = totalSolved

	c.JSON(http.StatusOK, profile)
}

func GetSolvedProblems(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := db.DB.Query(`SELECT problem_id FROM user_problem_status WHERE user_id = $1 AND status = 'ACCEPTED'`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch solved problems"})
		return
	}
	defer rows.Close()

	var solved []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err == nil {
			solved = append(solved, pid)
		}
	}

	c.JSON(http.StatusOK, gin.H{"solved_problems": solved})
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

type VerifyCollegeEmailOTPRequest struct {
	OTP   string `json:"otp" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func VerifyCollegeEmailOTP(c *gin.Context) {
	var req VerifyCollegeEmailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	key := fmt.Sprintf("verify_college:%d", userID)

	// Fetch from Redis
	storedVal, err := db.RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP expired or not found"})
		return
	}

	// Stored val is "email:otp"
	parts := strings.Split(storedVal, ":")
	if len(parts) != 2 || parts[0] != req.Email || parts[1] != req.OTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OTP or Email"})
		return
	}

	// Extract roll number from email (e.g., nc.sc.u4cse24012@...)
	rollNo := ""
	emailParts := strings.Split(req.Email, "@")
	if len(emailParts) > 0 {
		rollNo = strings.ToUpper(emailParts[0])
	}

	// Update DB
	_, err = db.DB.Exec(`UPDATE users SET college_email = $1, is_college_verified = true, roll_no = $2 WHERE id = $3`, req.Email, rollNo, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update college email"})
		return
	}

	// Delete from Redis
	db.RedisClient.Del(context.Background(), key)

	c.JSON(http.StatusOK, gin.H{"message": "College email verified successfully!"})
}

// -- Codeforces Verification Logic --

type InitCFRequest struct {
	Handle string `json:"handle" binding:"required"`
}

func InitCFVerification(c *gin.Context) {
	var req InitCFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Handle is required"})
		return
	}
	userID, _ := c.Get("user_id")

	// Generate a random 8-character string for CF verification
	verifyString := generateOTP() + generateOTP() // Reuse existing generateOTP (usually 6 digits, so this makes 12)
	
	_, err := db.DB.Exec(`UPDATE users SET cf_handle = $1, cf_verify_string = $2, is_cf_verified = false WHERE id = $3`, req.Handle, verifyString, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize verification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Initialized",
		"verify_string": verifyString,
	})
}

func VerifyCF(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var handle, verifyString string
	err := db.DB.QueryRow(`SELECT cf_handle, cf_verify_string FROM users WHERE id = $1`, userID).Scan(&handle, &verifyString)
	if err != nil || handle == "" || verifyString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CF verification not initialized. Please set a handle first."})
		return
	}

	// Make request to Codeforces API
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://codeforces.com/api/user.info?handles=" + handle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reach Codeforces API"})
		return
	}
	defer resp.Body.Close()

	var cfResp struct {
		Status string `json:"status"`
		Result []struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil || cfResp.Status != "OK" || len(cfResp.Result) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Codeforces handle or API error"})
		return
	}

	cfUser := cfResp.Result[0]
	
	// Check if the verifyString is in FirstName or LastName
	if !strings.Contains(cfUser.FirstName, verifyString) && !strings.Contains(cfUser.LastName, verifyString) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification string not found in Codeforces First Name or Last Name."})
		return
	}

	// If verified, update DB
	_, err = db.DB.Exec(`UPDATE users SET is_cf_verified = true WHERE id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save verification status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Codeforces account successfully verified!"})
}

func DisconnectCF(c *gin.Context) {
	userID, _ := c.Get("user_id")

	_, err := db.DB.Exec(`UPDATE users SET cf_handle = NULL, cf_verify_string = NULL, is_cf_verified = false WHERE id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Codeforces account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Codeforces account disconnected successfully!"})
}

// -- Profile Edit Logic --

type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// Get current email
	var currentEmail string
	err := db.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, userID).Scan(&currentEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Update Name immediately
	_, err = db.DB.Exec(`UPDATE users SET name = $1 WHERE id = $2`, req.Name, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile name"})
		return
	}

	if req.Email != currentEmail {
		// Start Email OTP Flow
		otp := generateOTP()
		key := fmt.Sprintf("email_update_otp:%d", userID)
		
		err := db.RedisClient.Set(context.Background(), key, req.Email+":"+otp, 10*time.Minute).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
			return
		}

		// Store pending email in DB
		_, err = db.DB.Exec(`UPDATE users SET pending_email = $1 WHERE id = $2`, req.Email, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pending email"})
			return
		}

		// Send Email
		body := fmt.Sprintf("Hello %s,\n\nYou requested to change your email address. Your OTP is: %s\n\nThis code expires in 10 minutes.", req.Name, otp)
		go mailer.SendEmail(req.Email, "NicheCP Email Update Verification", body)

		c.JSON(http.StatusOK, gin.H{"message": "Profile name updated. OTP sent to your new email for verification.", "require_otp": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully", "require_otp": false})
}

type VerifyEmailUpdateRequest struct {
	OTP string `json:"otp" binding:"required"`
}

func VerifyEmailUpdate(c *gin.Context) {
	var req VerifyEmailUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	key := fmt.Sprintf("email_update_otp:%d", userID)

	// Fetch from Redis
	storedVal, err := db.RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP expired or not found"})
		return
	}

	// Stored val is "email:otp"
	parts := strings.Split(storedVal, ":")
	if len(parts) != 2 || parts[1] != req.OTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OTP"})
		return
	}

	newEmail := parts[0]

	// Check if pending_email matches newEmail
	var pendingEmail *string
	err = db.DB.QueryRow(`SELECT pending_email FROM users WHERE id = $1`, userID).Scan(&pendingEmail)
	if err != nil || pendingEmail == nil || *pendingEmail != newEmail {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email mismatch or no pending email"})
		return
	}

	// Update DB
	_, err = db.DB.Exec(`UPDATE users SET email = $1, pending_email = NULL WHERE id = $2`, newEmail, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email. Email might already be taken."})
		return
	}

	// Delete from Redis
	db.RedisClient.Del(context.Background(), key)

	c.JSON(http.StatusOK, gin.H{"message": "Email updated successfully!"})
}

func UploadProfilePicture(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// Enforce 2MB size limit
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	// Parse the multipart form
	file, header, err := c.Request.FormFile("profile_picture")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is too large (max 2MB) or missing"})
		return
	}
	defer file.Close()

	// Check extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPG, PNG, and WebP images are allowed"})
		return
	}

	// Sniff MIME type securely
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file for validation"})
		return
	}
	fileType := http.DetectContentType(buff)
	if !strings.HasPrefix(fileType, "image/") || strings.Contains(fileType, "svg") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file format"})
		return
	}

	// Reset file pointer after read
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file upload"})
		return
	}

	// Create unique filename
	filename := fmt.Sprintf("dp_%d_%d%s", userID, time.Now().Unix(), ext)
	uploadPath := filepath.Join("uploads", filename)

	// Create file on disk
	out, err := os.Create(uploadPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file"})
		return
	}

	// Update DB with URL
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	picURL := backendURL + "/uploads/" + filename
	_, err = db.DB.Exec(`UPDATE users SET profile_picture_url = $1 WHERE id = $2`, picURL, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile picture in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile picture uploaded", "url": picURL})
}

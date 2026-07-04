package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Generate a random state string for CSRF protection
func generateStateOauthCookie(c *gin.Context) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	// Set cookie to expire in 20 minutes
	c.SetCookie("oauthstate", state, int(20*time.Minute.Seconds()), "/", "", false, true)
	return state
}

// GoogleLogin redirects the user to the Google OAuth consent screen
func GoogleLogin(c *gin.Context) {
	oauthState := generateStateOauthCookie(c)
	url := auth.GoogleOAuthConfig.AuthCodeURL(oauthState)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles the response from Google
func GoogleCallback(c *gin.Context) {
	oauthState, err := c.Cookie("oauthstate")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing oauth state cookie"})
		return
	}

	if c.Query("state") != oauthState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid oauth state"})
		return
	}

	code := c.Query("code")
	token, err := auth.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Code exchange failed"})
		return
	}

	// Fetch user info from Google
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed getting user info"})
		return
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed reading response body"})
		return
	}

	var googleUser struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(contents, &googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Google user info"})
		return
	}

	// Find or Create user in DB
	var userID int
	err = db.DB.QueryRow(`
		INSERT INTO users (name, email) 
		VALUES ($1, $2) 
		ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name 
		RETURNING id
	`, googleUser.Name, googleUser.Email).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync user to database"})
		return
	}

	// Generate our own JWT
	jwtToken, err := auth.GenerateToken(userID, googleUser.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	// For a simple SPA, we can set the JWT in an HTTP-only cookie, 
	// or redirect back to the frontend with the token in the URL fragment (hash).
	// We will redirect back to the frontend's home page (on port 3000) and pass the token securely.
	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/index.html?token="+jwtToken)
}

// --- Standard Authentication ---

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	RollNo   string `json:"roll_no" binding:"required"`
	Batch    string `json:"batch" binding:"required"`
	OTP      string `json:"otp" binding:"required,len=6"`
}

func RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify OTP
	key := fmt.Sprintf("register_otp:%s", req.Email)
	storedOTP, err := db.RedisClient.Get(context.Background(), key).Result()
	if err != nil || storedOTP != req.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP."})
		return
	}

	// Clear OTP after successful use
	db.RedisClient.Del(context.Background(), key)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var userID int
	err = db.DB.QueryRow(`
		INSERT INTO users (name, email, password_hash, roll_no, batch) 
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, req.Name, req.Email, string(hashedPassword), req.RollNo, req.Batch).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully!"})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID int
	var passwordHash *string
	err := db.DB.QueryRow(`SELECT id, password_hash FROM users WHERE email = $1`, req.Email).Scan(&userID, &passwordHash)
	if err != nil || passwordHash == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password. (OAuth users must sign in with Google)"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	jwtToken, err := auth.GenerateToken(userID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

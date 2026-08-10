package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func extractRollNumber(email string) string {
	if strings.HasSuffix(email, "amrita.edu") {
		parts := strings.Split(email, "@")
		return strings.ToUpper(parts[0])
	}
	return ""
}

// getCookieConfig determines the cookie configuration based on the environment
func getCookieConfig(c *gin.Context) (domain string, sameSite http.SameSite, isSecure bool) {
	host := c.Request.Host
	if hostHeader := c.GetHeader("X-Forwarded-Host"); hostHeader != "" {
		host = hostHeader
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	// Check if it's a local/development environment
	isLocal := host == "localhost" || host == "127.0.0.1" ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.") ||
		net.ParseIP(host) != nil

	// Determine if secure (HTTPS)
	isSecure = c.Request.TLS != nil ||
		c.GetHeader("X-Forwarded-Proto") == "https" ||
		strings.HasPrefix(c.GetHeader("X-Forwarded-Proto"), "https")

	// CRITICAL FIX: Domain handling
	if isLocal {
		// For localhost, don't set domain (host-only cookie)
		domain = ""
		// For local development without HTTPS, SameSite=Lax is fine
		sameSite = http.SameSiteLaxMode
	} else {
		// For production, use the environment variable
		domain = os.Getenv("COOKIE_DOMAIN")
		// If COOKIE_DOMAIN is not set, try to extract it from the host
		if domain == "" {
			// Extract domain from host (e.g., api.nichecp.app -> .nichecp.app)
			parts := strings.Split(host, ".")
			if len(parts) >= 2 {
				// For subdomains like api.nichecp.app
				if len(parts) >= 3 {
					domain = "." + strings.Join(parts[len(parts)-2:], ".")
				} else {
					domain = "." + host
				}
			}
		}
		// For cross-domain production, use SameSite=None with Secure
		// For same-domain, SameSite=Lax is fine
		sameSite = http.SameSiteLaxMode
	}

	// Override SameSite for production cross-domain if needed
	if !isLocal && os.Getenv("OAUTH_SAME_SITE") == "none" {
		sameSite = http.SameSiteNoneMode
		// Must be secure if SameSite=None
		isSecure = true
	}

	log.Printf("[Cookie Config] host=%s, domain=%s, isSecure=%v, sameSite=%v, isLocal=%v",
		host, domain, isSecure, sameSite, isLocal)

	return domain, sameSite, isSecure
}

// generateStateOauthCookie generates a secure state token and stores it in both a cookie and Redis.
// Redis serves as a reliable fallback when cookies fail due to cross-domain/SameSite issues.
func generateStateOauthCookie(c *gin.Context) string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	domain, sameSite, isSecure := getCookieConfig(c)

	log.Printf("[Cookie Set] state=%s, domain=%s, sameSite=%v, isSecure=%v, path=/",
		state[:10]+"...", domain, sameSite, isSecure)

	// Set the cookie with explicit path
	c.SetSameSite(sameSite)
	c.SetCookie(
		"oauthstate",
		state,
		int(20*time.Minute.Seconds()),
		"/",
		domain,
		isSecure,
		true, // HttpOnly
	)

	// Also store state in Redis as a fallback for cross-domain cookie failures.
	// The key is the state value itself; existence = valid.
	redisKey := fmt.Sprintf("oauth_state:%s", state)
	if err := db.RedisClient.Set(context.Background(), redisKey, "1", 20*time.Minute).Err(); err != nil {
		log.Printf("[OAuth Warning] Failed to store state in Redis: %v (cookie-only mode)", err)
	}

	return state
}

// GoogleLogin redirects the user to the Google OAuth consent screen
func GoogleLogin(c *gin.Context) {
	log.Printf("[OAuth Login] Starting Google login flow from host: %s", c.Request.Host)

	oauthState := generateStateOauthCookie(c)

	referer := c.Request.Referer()
	if referer != "" {
		domain, sameSite, isSecure := getCookieConfig(c)
		c.SetSameSite(sameSite)
		c.SetCookie("oauth_referer", referer, int(20*time.Minute.Seconds()), "/", domain, isSecure, true)
	}

	domain, _, isSecure := getCookieConfig(c)
	_ = domain

	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	// Auto-construct redirect URL from request host when:
	// 1. GOOGLE_REDIRECT_URL is not set, OR
	// 2. Request is local (use current host for easy local dev), OR
	// 3. GOOGLE_REDIRECT_URL points to localhost but we're serving production traffic
	isLocalRequest := strings.Contains(c.Request.Host, "localhost") || strings.Contains(c.Request.Host, "127.0.0.1")
	isLocalRedirect := strings.Contains(redirectURL, "localhost") || strings.Contains(redirectURL, "127.0.0.1")
	if redirectURL == "" || isLocalRequest || (isLocalRedirect && !isLocalRequest) {
		scheme := "http"
		if isSecure {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s/api/auth/google/callback", scheme, c.Request.Host)
		log.Printf("[OAuth Login] Auto-constructed redirect URL: %s", redirectURL)
	}

	conf := *auth.GoogleOAuthConfig
	conf.RedirectURL = redirectURL

	authURL := conf.AuthCodeURL(oauthState)
	log.Printf("[OAuth Login] Redirecting to Google: %s", authURL)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// GoogleCallback handles the response from Google
func GoogleCallback(c *gin.Context) {
	log.Printf("[OAuth Callback] Received callback from Google")
	log.Printf("[OAuth Debug] Host=%s, X-Forwarded-Host=%s, X-Forwarded-Proto=%s",
		c.Request.Host,
		c.GetHeader("X-Forwarded-Host"),
		c.GetHeader("X-Forwarded-Proto"))

	// Log all cookies for debugging
	cookies := c.Request.Cookies()
	log.Printf("[OAuth Debug] Total cookies in request: %d", len(cookies))
	for _, cookie := range cookies {
		if cookie.Name == "oauthstate" {
			log.Printf("[OAuth Debug] Found oauthstate cookie: %s", cookie.Value[:10]+"...")
		}
	}

	domain, sameSite, isSecure := getCookieConfig(c)

	// Get the state from the query parameter (always present from Google)
	queryState := c.Query("state")
	if queryState == "" {
		log.Printf("[OAuth Error] Missing state query parameter from Google redirect")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state parameter"})
		return
	}

	// Try to validate state via cookie first (preferred, proves same browser)
	oauthState, cookieErr := c.Cookie("oauthstate")

	// Clear the state cookie after retrieval to prevent reuse
	c.SetSameSite(sameSite)
	c.SetCookie("oauthstate", "", -1, "/", domain, isSecure, true)

	stateVerified := false

	if cookieErr == nil && oauthState == queryState {
		// Cookie-based verification succeeded
		log.Printf("[OAuth Debug] State verified via cookie")
		stateVerified = true
	} else {
		if cookieErr != nil {
			log.Printf("[OAuth Warning] Cookie missing (err: %v), falling back to Redis verification", cookieErr)
		} else {
			log.Printf("[OAuth Warning] Cookie state mismatch (cookie: %s, query: %s), trying Redis", oauthState[:10]+"...", queryState[:10]+"...")
		}

		// Fallback: verify state exists in Redis (proves it was generated by our server)
		redisKey := fmt.Sprintf("oauth_state:%s", queryState)
		exists, err := db.RedisClient.Exists(context.Background(), redisKey).Result()
		if err != nil {
			log.Printf("[OAuth Error] Redis state check failed: %v", err)
		} else if exists > 0 {
			log.Printf("[OAuth Debug] State verified via Redis")
			stateVerified = true
			// Delete the key to prevent reuse
			db.RedisClient.Del(context.Background(), redisKey)
		}
	}

	if !stateVerified {
		log.Printf("[OAuth Error] State verification failed. Cookie err: %v, Query state: %s, Cookie count: %d",
			cookieErr, queryState[:10]+"...", len(c.Request.Cookies()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid oauth state - possible CSRF attack",
			"debug": fmt.Sprintf("Host: %s, X-Forwarded-Host: %s, Cookie Count: %d",
				c.Request.Host,
				c.GetHeader("X-Forwarded-Host"),
				len(c.Request.Cookies())),
			"solution": "Clear your cookies and try logging in again",
		})
		return
	}

	// Also clean up the Redis key if cookie verification was used
	if cookieErr == nil {
		redisKey := fmt.Sprintf("oauth_state:%s", queryState)
		db.RedisClient.Del(context.Background(), redisKey)
	}

	log.Printf("[OAuth Debug] State verification successful")

	// Get the redirect URL (must match what was sent to Google in GoogleLogin)
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	isLocalRequest := strings.Contains(c.Request.Host, "localhost") || strings.Contains(c.Request.Host, "127.0.0.1")
	isLocalRedirect := strings.Contains(redirectURL, "localhost") || strings.Contains(redirectURL, "127.0.0.1")
	if redirectURL == "" || isLocalRequest || (isLocalRedirect && !isLocalRequest) {
		scheme := "http"
		if isSecure {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s/api/auth/google/callback", scheme, c.Request.Host)
	}

	conf := *auth.GoogleOAuthConfig
	conf.RedirectURL = redirectURL

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		log.Printf("[OAuth Error] Missing code parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code parameter"})
		return
	}

	log.Printf("[OAuth Debug] Exchanging code for token...")
	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("[OAuth Error] Code exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Code exchange failed"})
		return
	}

	log.Printf("[OAuth Debug] Token exchange successful")

	// Fetch user info from Google
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		log.Printf("[OAuth Error] Failed getting user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed getting user info"})
		return
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("[OAuth Error] Failed reading response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed reading response body"})
		return
	}

	var googleUser struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(contents, &googleUser); err != nil {
		log.Printf("[OAuth Error] Failed to parse Google user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Google user info"})
		return
	}

	log.Printf("[OAuth Debug] Google user: email=%s, name=%s", googleUser.Email, googleUser.Name)

	// Auto-extract roll number if amrita.edu
	rollNo := extractRollNumber(googleUser.Email)

	// Find or Create user in DB
	var userID int
	isVerified := rollNo != ""
	err = db.DB.QueryRow(`
		INSERT INTO users (name, email, roll_no, is_college_verified) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (email) DO UPDATE SET 
			name = EXCLUDED.name, 
			roll_no = COALESCE(users.roll_no, EXCLUDED.roll_no),
			is_college_verified = EXCLUDED.is_college_verified
		RETURNING id
	`, googleUser.Name, googleUser.Email, rollNo, isVerified).Scan(&userID)

	if err != nil {
		log.Printf("[OAuth Error] Failed to sync user to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync user to database"})
		return
	}

	log.Printf("[OAuth Debug] User synced to DB with ID: %d", userID)

	// Generate our own JWT
	jwtToken, err := auth.GenerateToken(userID, googleUser.Email)
	if err != nil {
		log.Printf("[OAuth Error] Failed to generate JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	// Get frontend URL for redirect
	frontendURL := os.Getenv("FRONTEND_URL")
	if ref, err := c.Cookie("oauth_referer"); err == nil && ref != "" {
		if parsed, err := url.Parse(ref); err == nil {
			frontendURL = parsed.Scheme + "://" + parsed.Host
		}
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// Clear the referer cookie
	c.SetSameSite(sameSite)
	c.SetCookie("oauth_referer", "", -1, "/", domain, isSecure, true)

	// Redirect back to frontend with token
	redirectTarget := frontendURL + "/#token=" + jwtToken
	log.Printf("[OAuth Debug] Redirecting to: %s", redirectTarget)

	c.Redirect(http.StatusTemporaryRedirect, redirectTarget)
}

// --- Standard Authentication ---

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
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

	rollNo := extractRollNumber(req.Email)

	var userID int
	err = db.DB.QueryRow(`
		INSERT INTO users (name, email, password_hash, roll_no) 
		VALUES ($1, $2, $3, $4) RETURNING id
	`, req.Name, req.Email, string(hashedPassword), rollNo).Scan(&userID)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		} else {
			log.Printf("[Register Error] Failed to register user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
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

	// Auto-update roll number if missing
	rollNo := extractRollNumber(req.Email)
	if rollNo != "" {
		db.DB.Exec("UPDATE users SET roll_no = $1 WHERE id = $2 AND (roll_no IS NULL OR roll_no = '')", rollNo, userID)
	}

	jwtToken, err := auth.GenerateToken(userID, req.Email)
	if err != nil {
		log.Printf("[Login Error] Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}
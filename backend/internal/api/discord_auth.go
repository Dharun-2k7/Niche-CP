package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

// DiscordLogin redirects the user to the Discord OAuth consent screen
func DiscordLogin(c *gin.Context) {
	log.Printf("[Discord OAuth] Starting Discord login flow")

	oauthState := generateStateOauthCookie(c)

	referer := c.Request.Referer()
	if referer != "" {
		domain, sameSite, isSecure := getCookieConfig(c)
		c.SetSameSite(sameSite)
		c.SetCookie("oauth_referer", referer, int(20*time.Minute.Seconds()), "/", domain, isSecure, true)
	}

	_, _, isSecure := getCookieConfig(c)

	redirectURL := os.Getenv("DISCORD_REDIRECT_URL")
	isLocalRequest := strings.Contains(c.Request.Host, "localhost") || strings.Contains(c.Request.Host, "127.0.0.1")
	isLocalRedirect := strings.Contains(redirectURL, "localhost") || strings.Contains(redirectURL, "127.0.0.1")
	if redirectURL == "" || isLocalRequest || (isLocalRedirect && !isLocalRequest) {
		scheme := "http"
		if isSecure {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s/api/auth/discord/callback", scheme, c.Request.Host)
	}

	conf := *auth.DiscordOAuthConfig
	conf.RedirectURL = redirectURL

	authURL := conf.AuthCodeURL(oauthState)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// DiscordCallback handles the response from Discord
func DiscordCallback(c *gin.Context) {
	log.Printf("[Discord OAuth] Received callback from Discord")

	domain, sameSite, isSecure := getCookieConfig(c)

	queryState := c.Query("state")
	if queryState == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state parameter"})
		return
	}

	// Validate state via cookie or Redis (reuse same logic as Google)
	oauthState, cookieErr := c.Cookie("oauthstate")
	c.SetSameSite(sameSite)
	c.SetCookie("oauthstate", "", -1, "/", domain, isSecure, true)

	stateVerified := false
	if cookieErr == nil && oauthState == queryState {
		stateVerified = true
	} else {
		redisKey := fmt.Sprintf("oauth_state:%s", queryState)
		exists, err := db.RedisClient.Exists(context.Background(), redisKey).Result()
		if err == nil && exists > 0 {
			stateVerified = true
			db.RedisClient.Del(context.Background(), redisKey)
		}
	}

	if !stateVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid oauth state"})
		return
	}

	if cookieErr == nil {
		redisKey := fmt.Sprintf("oauth_state:%s", queryState)
		db.RedisClient.Del(context.Background(), redisKey)
	}

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code parameter"})
		return
	}

	redirectURL := os.Getenv("DISCORD_REDIRECT_URL")
	isLocalRequest := strings.Contains(c.Request.Host, "localhost") || strings.Contains(c.Request.Host, "127.0.0.1")
	isLocalRedirect := strings.Contains(redirectURL, "localhost") || strings.Contains(redirectURL, "127.0.0.1")
	if redirectURL == "" || isLocalRequest || (isLocalRedirect && !isLocalRequest) {
		scheme := "http"
		if isSecure {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s/api/auth/discord/callback", scheme, c.Request.Host)
	}

	conf := *auth.DiscordOAuthConfig
	conf.RedirectURL = redirectURL

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("[Discord OAuth Error] Code exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Code exchange failed"})
		return
	}

	// Fetch user info from Discord
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed getting Discord user info"})
		return
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed reading response"})
		return
	}

	var discordUser struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Email         string `json:"email"`
		GlobalName    string `json:"global_name"`
	}
	if err := json.Unmarshal(contents, &discordUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Discord user info"})
		return
	}

	log.Printf("[Discord OAuth] Discord user: id=%s, username=%s, email=%s", discordUser.ID, discordUser.Username, discordUser.Email)

	displayName := discordUser.GlobalName
	if displayName == "" {
		displayName = discordUser.Username
	}

	// Try to find user by discord_id first
	var userID int
	err = db.DB.QueryRow(`SELECT id FROM users WHERE discord_id = $1`, discordUser.ID).Scan(&userID)
	if err == nil {
		// User found by discord_id - update username and log in
		db.DB.Exec(`UPDATE users SET discord_username = $1 WHERE id = $2`, discordUser.Username, userID)
		var email string
		db.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, userID).Scan(&email)

		jwtToken, err := auth.GenerateToken(userID, email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
			return
		}

		frontendURL := getFrontendURL(c, domain, isSecure)
		c.SetSameSite(sameSite)
		c.SetCookie("oauth_referer", "", -1, "/", domain, isSecure, true)
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/#token="+jwtToken)
		return
	}

	// Try to find user by email (link Discord to existing account)
	if discordUser.Email != "" {
		err = db.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, discordUser.Email).Scan(&userID)
		if err == nil {
			// Link Discord to existing account
			db.DB.Exec(`UPDATE users SET discord_id = $1, discord_username = $2 WHERE id = $3`,
				discordUser.ID, discordUser.Username, userID)

			jwtToken, err := auth.GenerateToken(userID, discordUser.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
				return
			}

			frontendURL := getFrontendURL(c, domain, isSecure)
			c.SetSameSite(sameSite)
			c.SetCookie("oauth_referer", "", -1, "/", domain, isSecure, true)
			c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/#token="+jwtToken)
			return
		}
	}

	// New user - create account
	email := discordUser.Email
	if email == "" {
		email = discordUser.ID + "@discord.user" // Fallback for users without email scope
	}

	rollNo := extractRollNumber(email)
	isVerified := rollNo != ""

	err = db.DB.QueryRow(`
		INSERT INTO users (name, email, discord_id, discord_username, roll_no, is_college_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (email) DO UPDATE SET
			discord_id = EXCLUDED.discord_id,
			discord_username = EXCLUDED.discord_username
		RETURNING id
	`, displayName, email, discordUser.ID, discordUser.Username, rollNo, isVerified).Scan(&userID)
	if err != nil {
		log.Printf("[Discord OAuth Error] Failed to sync user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	jwtToken, err := auth.GenerateToken(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	frontendURL := getFrontendURL(c, domain, isSecure)
	c.SetSameSite(sameSite)
	c.SetCookie("oauth_referer", "", -1, "/", domain, isSecure, true)
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/#token="+jwtToken)
}

// getFrontendURL gets the frontend URL from referer cookie or env
func getFrontendURL(c *gin.Context, domain string, isSecure bool) string {
	frontendURL := os.Getenv("FRONTEND_URL")
	if ref, err := c.Cookie("oauth_referer"); err == nil && ref != "" {
		if parsed, err := url.Parse(ref); err == nil {
			frontendURL = parsed.Scheme + "://" + parsed.Host
		}
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	return frontendURL
}

// LinkDiscord links a Discord account to an authenticated user
func LinkDiscord(c *gin.Context) {
	log.Printf("[Discord Link] Starting Discord account linking")

	oauthState := generateStateOauthCookie(c)

	// Store the fact that this is a linking operation (not login)
	domain, sameSite, isSecure := getCookieConfig(c)
	c.SetSameSite(sameSite)
	c.SetCookie("discord_link_mode", "true", int(20*time.Minute.Seconds()), "/", domain, isSecure, true)

	// Store the JWT token so we know which user is linking
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required to link Discord"})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	c.SetSameSite(sameSite)
	c.SetCookie("discord_link_token", tokenStr, int(20*time.Minute.Seconds()), "/", domain, isSecure, true)

	redirectURL := os.Getenv("DISCORD_REDIRECT_URL")
	if redirectURL == "" {
		scheme := "http"
		if isSecure {
			scheme = "https"
		}
		redirectURL = fmt.Sprintf("%s://%s/api/auth/discord/callback", scheme, c.Request.Host)
	}

	conf := *auth.DiscordOAuthConfig
	conf.RedirectURL = redirectURL

	authURL := conf.AuthCodeURL(oauthState)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// UnlinkDiscord removes Discord association from user's account
func UnlinkDiscord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	_, err := db.DB.Exec(`UPDATE users SET discord_id = NULL, discord_username = NULL WHERE id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink Discord"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Discord account unlinked successfully"})
}

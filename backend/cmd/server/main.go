package main

import (
	"log"
	"os"
	"strings"

	"github.com/Dharun-2k7/online-coding-platform/internal/api"
	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/Dharun-2k7/online-coding-platform/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default environment variables.")
	}

	db.InitPostgres()
	db.InitRedis()
	auth.InitOAuth()
	judge.InitSemaphore()
	api.StartContestStateTicker()

	// Setup Router
	r := gin.Default()

	// Log all incoming requests (helpful for debugging)
	r.Use(func(c *gin.Context) {
		log.Printf("[Request] %s %s | Host: %s | Cookies: %d", 
			c.Request.Method, 
			c.Request.URL.Path, 
			c.Request.Host,
			len(c.Request.Cookies()))
		c.Next()
	})

	// CORS middleware with proper configuration
	r.Use(func(c *gin.Context) {
		allowedOrigin := os.Getenv("FRONTEND_URL")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000" // Default for local dev
		}
		
		// For local development, also allow any localhost port
		origin := c.GetHeader("Origin")
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			allowedOrigin = origin
		}
		
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Cookie debugging middleware for OAuth routes
	r.Use(func(c *gin.Context) {
		if strings.Contains(c.Request.URL.Path, "/auth/google") {
			log.Printf("[OAuth Debug] Path: %s", c.Request.URL.Path)
			log.Printf("[OAuth Debug] Host: %s", c.Request.Host)
			log.Printf("[OAuth Debug] Cookies: %+v", c.Request.Cookies())
			log.Printf("[OAuth Debug] Headers: X-Forwarded-Host=%s, X-Forwarded-Proto=%s", 
				c.GetHeader("X-Forwarded-Host"), 
				c.GetHeader("X-Forwarded-Proto"))
		}
		c.Next()
	})

	// Serve static uploads
	r.Static("/uploads", "./uploads")

	// Public Routes
	r.GET("/health", api.HealthCheck)
	r.POST("/api/run", api.RunCode) // Allow unauthenticated manual runs

	// Auth Routes - OAuth needs to be accessible without auth
	r.GET("/api/auth/google/login", api.GoogleLogin)
	r.GET("/api/auth/google/callback", api.GoogleCallback)
	r.GET("/api/auth/discord/login", api.DiscordLogin)
	r.GET("/api/auth/discord/callback", api.DiscordCallback)
	r.POST("/api/auth/register", api.RegisterUser)
	r.POST("/api/auth/login", api.LoginUser)

	// New Auth Flows
	r.POST("/api/auth/otp/send", api.SendOTP)
	r.POST("/api/auth/otp/verify", api.VerifyOTP)
	r.POST("/api/auth/register-otp/send", api.SendRegisterOTP)
	r.POST("/api/auth/register-otp/verify", api.RegisterUser)
	r.POST("/api/auth/password/forgot", api.ForgotPassword)
	r.POST("/api/auth/password/reset", api.ResetPassword)

	// Problems Route
	r.GET("/api/problems", api.GetAllProblems)
	r.GET("/api/problems/:id", api.GetProblem)
	r.GET("/api/contests", api.GetAllContests)
	r.GET("/api/contests/:id/leaderboard", api.GetContestLeaderboard)

	// Public Homepage Routes
	r.GET("/api/public/upcoming-contests", api.GetUpcomingContests)
	r.GET("/api/public/recent-problems", api.GetRecentProblems)

	// Protected Routes
	protected := r.Group("/api")
	protected.Use(middleware.RequireAuth())

	// Contest Lifecycle Routes
	contestProtected := r.Group("/api/contests")
	contestProtected.Use(middleware.RequireContestLifecycle())
	{
		contestProtected.GET("/:id/problems", api.GetContestProblems)
		contestProtected.GET("/:id/submissions", api.GetContestSubmissions)
		contestProtected.GET("/:id/my-submissions", api.GetMyContestSubmissions)
		contestProtected.GET("/:id/access", api.CheckContestAccess)
	}
	{
		protected.POST("/submit", api.SubmitCode)
		protected.GET("/submissions/:id", api.GetSubmissionStatus)
		protected.GET("/profile", api.GetProfile)
		protected.GET("/profile/stats", api.GetUserStats)
		protected.GET("/profile/solved", api.GetSolvedProblems)
		protected.PUT("/profile", api.UpdateProfile)
		protected.POST("/profile/upload-dp", api.UploadProfilePicture)
		protected.POST("/profile/verify-college-email", api.VerifyCollegeEmail)
		protected.POST("/profile/verify-otp", api.VerifyCollegeEmailOTP)
		protected.POST("/profile/email/verify-existing", api.VerifyExistingEmailOTP)
		protected.POST("/profile/email/verify-new", api.VerifyNewEmailOTP)
		protected.POST("/profile/cf/init", api.InitCFVerification)
		protected.POST("/profile/cf/verify", api.VerifyCF)
		protected.POST("/profile/cf/disconnect", api.DisconnectCF)
		protected.GET("/contests/my-registrations", api.GetMyRegistrations)
		protected.POST("/contests/:id/register", api.RegisterForContest)
		protected.POST("/contests/:id/log", api.LogContestViolation)
		protected.GET("/profile/discord/link", api.LinkDiscord)
		protected.POST("/profile/discord/unlink", api.UnlinkDiscord)
	}

	// Admin Routes
	admin := r.Group("/api/admin")
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/stats", api.GetAdminStats)
		admin.GET("/users", api.GetAllUsers)
		admin.POST("/users/permissions", api.UpdateUserPermissions)
		admin.POST("/promote", api.PromoteToAdmin)
		admin.POST("/demote", api.DemoteFromAdmin)
		admin.POST("/problems", api.CreateProblem)
		admin.GET("/problems/:id", api.GetAdminProblem)
		admin.PUT("/problems/:id", api.UpdateProblem)
		admin.POST("/contests", api.CreateContest)
		admin.PUT("/contests/:id", api.UpdateContest)
		admin.DELETE("/contests/:id", api.DeleteContest)
		admin.GET("/contests/:id/details", api.GetContestDetails)
		admin.GET("/contests/:id/violations", api.GetContestViolations)
	}

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Starting server on :%s", port)
	log.Printf("OAuth Redirect URL: %s", os.Getenv("GOOGLE_REDIRECT_URL"))
	log.Printf("Frontend URL: %s", os.Getenv("FRONTEND_URL"))
	log.Printf("Cookie Domain: %s", os.Getenv("COOKIE_DOMAIN"))
	
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
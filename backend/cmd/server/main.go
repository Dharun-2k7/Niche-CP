package main

import (
	"log"

	"github.com/Dharun-2k7/online-coding-platform/internal/api"
	"github.com/Dharun-2k7/online-coding-platform/internal/auth"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default environment variables.")
	}

	// Initialize Postgres and Redis
	db.InitPostgres()
	db.InitRedis()
	auth.InitOAuth()

	// Setup Router
	r := gin.Default()

	// Basic CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve static uploads
	r.Static("/uploads", "./uploads")

	// Public Routes
	r.GET("/health", api.HealthCheck)
	r.GET("/api/submissions/:id", api.GetSubmissionStatus)
	r.POST("/api/run", api.RunCode) // Allow unauthenticated manual runs

	// Auth Routes
	r.GET("/api/auth/google/login", api.GoogleLogin)
	r.GET("/api/auth/google/callback", api.GoogleCallback)
	r.POST("/api/auth/register", api.RegisterUser)
	r.POST("/api/auth/login", api.LoginUser)
	
	// New Auth Flows
	r.POST("/api/auth/otp/send", api.SendOTP)
	r.POST("/api/auth/otp/verify", api.VerifyOTP)
	r.POST("/api/auth/register-otp/send", api.SendRegisterOTP)
	r.POST("/api/auth/password/forgot", api.ForgotPassword)
	r.POST("/api/auth/password/reset", api.ResetPassword)

	// Protected Routes
	protected := r.Group("/api")
	protected.Use(middleware.RequireAuth())
	{
		protected.POST("/submit", api.SubmitCode)
		protected.GET("/profile", api.GetProfile)
		protected.PUT("/profile", api.UpdateProfile)
		protected.POST("/profile/upload-dp", api.UploadProfilePicture)
		protected.POST("/profile/verify-college-email", api.VerifyCollegeEmail)
		protected.POST("/profile/verify-otp", api.VerifyCollegeEmailOTP)
		protected.POST("/profile/cf/init", api.InitCFVerification)
		protected.POST("/profile/cf/verify", api.VerifyCF)
	}

	// Start Server
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

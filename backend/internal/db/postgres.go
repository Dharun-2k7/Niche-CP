package db

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitPostgres() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Println("WARNING: DATABASE_URL is not set. Database connection will fail.")
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open Postgres connection: %v", err)
	}

	// Retry loop: Postgres container may not be ready yet in Docker
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		err = DB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for Postgres (attempt %d/%d): %v", i, maxRetries, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to Postgres after %d attempts: %v", maxRetries, err)
	}

	log.Println("Successfully connected to Postgres!")
}

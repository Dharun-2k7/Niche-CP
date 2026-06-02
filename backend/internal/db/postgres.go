package db

import (
	"database/sql"
	"log"
	"os"

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

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping Postgres: %v", err)
	}

	log.Println("Successfully connected to Postgres!")
}

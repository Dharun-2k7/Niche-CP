package main

import (
	"fmt"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.InitPostgres()
	rows, _ := db.DB.Query("SELECT id, status FROM submissions WHERE id >= 1000")
	for rows.Next() {
		var id int
		var status string
		rows.Scan(&id, &status)
		fmt.Printf("Sub %d: %s\n", id, status)
	}
}

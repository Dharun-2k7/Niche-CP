package main

import (
	"context"
	"fmt"
	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.InitRedis()
	count, _ := db.RedisClient.LLen(context.Background(), "submissions_queue").Result()
	fmt.Printf("Queue length: %d\n", count)
}

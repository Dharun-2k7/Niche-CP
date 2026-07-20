package judge_test

import (
	"context"
	"testing"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/joho/godotenv"
)

func TestExecutionSemaphoreHeartbeat(t *testing.T) {
	_ = godotenv.Load("../../.env")
	db.InitRedis()
	
	ctx := context.Background()
	// Clear semaphore set before test
	db.RedisClient.Del(ctx, "nichecp:execution_semaphore")
	
	// Test 1: Acquire token with 4 seconds lease, check expiry
	token, err := judge.AcquireExecutionToken(ctx, 4, 4*time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire token: %v", err)
	}
	
	// Ensure token exists
	count, _ := db.RedisClient.ZCard(ctx, "nichecp:execution_semaphore").Result()
	if count != 1 {
		t.Errorf("Expected 1 token, got %d", count)
	}
	
	// Wait 5 seconds, token should expire
	time.Sleep(5 * time.Second)
	
	// Try to acquire again, should succeed and clean up expired token
	_, err = judge.AcquireExecutionToken(ctx, 4, 4*time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire second token: %v", err)
	}
	
	// Check count, should still be 1 because first token was expired and removed
	count, _ = db.RedisClient.ZCard(ctx, "nichecp:execution_semaphore").Result()
	if count != 1 {
		t.Errorf("Expected 1 token after expiry cleanup, got %d", count)
	}
	
	db.RedisClient.Del(ctx, "nichecp:execution_semaphore")

	// Test 2: Heartbeat keeps token alive
	token, err = judge.AcquireExecutionToken(ctx, 4, 4*time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire token for heartbeat test: %v", err)
	}
	
	executionCtx, cancel := context.WithCancel(context.Background())
	judge.KeepAliveToken(executionCtx, token, 4*time.Second)
	
	// Wait 6 seconds, the token should NOT expire because heartbeat renews it every 2s
	time.Sleep(6 * time.Second)
	
	// Try to acquire 4 more tokens (max is 4), the 5th should block/fail if our token is still alive
	// We'll just check if our token is still in the set
	score, err := db.RedisClient.ZScore(ctx, "nichecp:execution_semaphore", token).Result()
	if err != nil {
		t.Errorf("Token was leaked/expired despite heartbeat: %v", err)
	} else {
		t.Logf("Token successfully kept alive. Current score: %f", score)
	}
	
	// Test 3: Cancellation stops heartbeat
	cancel() // Stop heartbeat
	
	// Wait 5 seconds for it to naturally expire now that heartbeat stopped
	time.Sleep(5 * time.Second)
	
	// The token should be cleaned up on next acquisition
	_, _ = judge.AcquireExecutionToken(ctx, 4, 4*time.Second)
	_, err = db.RedisClient.ZScore(ctx, "nichecp:execution_semaphore", token).Result()
	if err == nil {
		t.Errorf("Token should have expired after heartbeat cancellation, but it still exists")
	} else {
		t.Logf("Token naturally expired after cancellation as expected")
	}
	
	db.RedisClient.Del(ctx, "nichecp:execution_semaphore")
}

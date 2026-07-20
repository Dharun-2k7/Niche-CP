package judge

import (
	"context"
	"fmt"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/google/uuid"
)

const executionSemaphoreKey = "nichecp:execution_semaphore"

// acquireLuaScript adds a token to a ZSET with the current timestamp as score,
// but only if the set has fewer than maxExecutions elements after removing expired ones.
const acquireLuaScript = `
local key = KEYS[1]
local max = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local token = ARGV[3]
local lease_ms = tonumber(ARGV[4])

-- Remove expired leases
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - lease_ms)

local count = redis.call('ZCARD', key)
if count < max then
    redis.call('ZADD', key, now, token)
    return 1
else
    return 0
end
`

// AcquireExecutionToken attempts to acquire an execution slot, polling until context expires.
// Returns a tokenID to be passed to ReleaseExecutionToken on success.
func AcquireExecutionToken(ctx context.Context, maxExecutions int, leaseDuration time.Duration) (string, error) {
	token := uuid.New().String()
	leaseMs := leaseDuration.Milliseconds()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Initial attempt without waiting
	now := time.Now().UnixMilli()
	res, err := db.RedisClient.Eval(ctx, acquireLuaScript, []string{executionSemaphoreKey}, maxExecutions, now, token, leaseMs).Result()
	if err != nil {
		return "", fmt.Errorf("failed to execute lua script: %w", err)
	}
	if res.(int64) == 1 {
		return token, nil
	}

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for execution capacity")
		case <-ticker.C:
			now := time.Now().UnixMilli()
			res, err := db.RedisClient.Eval(ctx, acquireLuaScript, []string{executionSemaphoreKey}, maxExecutions, now, token, leaseMs).Result()
			if err != nil {
				return "", fmt.Errorf("failed to execute lua script: %w", err)
			}
			if res.(int64) == 1 {
				return token, nil
			}
		}
	}
}

// ReleaseExecutionToken removes the token from the semaphore.
func ReleaseExecutionToken(ctx context.Context, token string) {
	// Use a fresh context with timeout to guarantee release even if original context was cancelled
	releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	_ = db.RedisClient.ZRem(releaseCtx, executionSemaphoreKey, token).Err()
}

// KeepAliveToken periodically updates the token's lease time to prevent expiration during long executions.
// It stops automatically when the provided context is canceled.
func KeepAliveToken(ctx context.Context, token string, leaseDuration time.Duration) {
	// Ping halfway through the lease duration, minimum 5 seconds
	interval := leaseDuration / 2
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := float64(time.Now().UnixMilli())
				// ZADD XX updates score only if the element already exists
				script := `
				local key = KEYS[1]
				local score = tonumber(ARGV[1])
				local token = ARGV[2]
				
				local score_exists = redis.call('ZSCORE', key, token)
				if score_exists then
					redis.call('ZADD', key, score, token)
				end
				`
				db.RedisClient.Eval(context.Background(), script, []string{executionSemaphoreKey}, now, token)
			}
		}
	}()
}

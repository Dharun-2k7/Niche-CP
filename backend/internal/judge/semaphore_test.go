package judge_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
)

func TestExecutionSemaphore(t *testing.T) {
	// Setup max workers to 2 for testing
	os.Setenv("MAX_EXECUTION_WORKERS", "2")
	judge.InitSemaphore()

	ctx := context.Background()

	// Test 1: Acquire tokens up to capacity
	err := judge.AcquireExecutionToken(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire first token: %v", err)
	}

	err = judge.AcquireExecutionToken(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire second token: %v", err)
	}

	// Test 2: Acquiring beyond capacity should timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = judge.AcquireExecutionToken(timeoutCtx)
	if err == nil {
		t.Fatalf("Expected timeout when acquiring beyond capacity, but succeeded")
	}

	// Test 3: Release token and acquire again
	judge.ReleaseExecutionToken()
	
	err = judge.AcquireExecutionToken(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire token after releasing one: %v", err)
	}
    
	// Clean up
	judge.ReleaseExecutionToken()
	judge.ReleaseExecutionToken()
}

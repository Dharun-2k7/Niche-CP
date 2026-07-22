package judge

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"os"
)

var (
	localSemaphore chan struct{}
)

// InitSemaphore initializes the local package-level Go channel semaphore.
// It uses the MAX_EXECUTION_WORKERS env var, defaulting to 4.
// This enforces a strict concurrency limit on a single VM without the complexity of Redis.
func InitSemaphore() {
	maxWorkersStr := os.Getenv("MAX_EXECUTION_WORKERS")
	maxWorkers := 4
	if maxWorkersStr != "" {
		if parsed, err := strconv.Atoi(maxWorkersStr); err == nil && parsed > 0 {
			maxWorkers = parsed
		}
	}
	localSemaphore = make(chan struct{}, maxWorkers)
	log.Printf("Initialized local Go channel semaphore with capacity: %d", maxWorkers)
}

// AcquireExecutionToken acquires an execution slot using the local buffered channel.
// Blocks until a slot is available or the context expires.
func AcquireExecutionToken(ctx context.Context) error {
	if localSemaphore == nil {
		return fmt.Errorf("local semaphore not initialized, call InitSemaphore() first")
	}

	select {
	case localSemaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for execution capacity")
	}
}

// ReleaseExecutionToken frees the execution slot back to the local channel.
func ReleaseExecutionToken() {
	if localSemaphore != nil {
		select {
		case <-localSemaphore:
			// successfully released
		default:
			log.Println("[WARNING] ReleaseExecutionToken called but semaphore channel was empty!")
		}
	}
}

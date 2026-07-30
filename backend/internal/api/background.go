package api

import (
	"log"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
)

// StartContestStateTicker starts a background goroutine to periodically update contest statuses.
func StartContestStateTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			<-ticker.C
			updateContestStates()
		}
	}()
	// Also run once on startup
	go updateContestStates()
}

func updateContestStates() {
	now := time.Now().UTC()

	// Update to UPCOMING (e.g. if we want to distinguish CREATED and UPCOMING, say UPCOMING is < 24h away)
	// For now, let's just do CREATED -> RUNNING -> ENDED. 
	// Wait, UPCOMING could mean registration is open.
	// We'll update RUNNING
	_, err := db.DB.Exec(`
		UPDATE contests 
		SET status = 'RUNNING' 
		WHERE status IN ('CREATED', 'UPCOMING') 
		  AND start_time <= $1 
		  AND end_time > $1
	`, now)
	if err != nil {
		log.Printf("Error updating contests to RUNNING: %v", err)
	}

	// Update to ENDED
	_, err = db.DB.Exec(`
		UPDATE contests 
		SET status = 'ENDED' 
		WHERE status = 'RUNNING' 
		  AND end_time <= $1
	`, now)
	if err != nil {
		log.Printf("Error updating contests to ENDED: %v", err)
	}
}

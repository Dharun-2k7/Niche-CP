package api

import (
	"log"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
)

// StartContestStateTicker starts a background goroutine to periodically update contest statuses.
func StartContestStateTicker() {
	ticker := time.NewTicker(10 * time.Second)
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

	// Update to RUNNING when start_time <= now < end_time
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

	// Update to ENDED when end_time <= now
	_, err = db.DB.Exec(`
		UPDATE contests 
		SET status = 'ENDED' 
		WHERE status IN ('CREATED', 'UPCOMING', 'RUNNING') 
		  AND end_time <= $1
	`, now)
	if err != nil {
		log.Printf("Error updating contests to ENDED: %v", err)
	}
}

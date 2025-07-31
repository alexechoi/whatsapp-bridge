package main

import (
	"net/http"
	"time"
)

var isMainAppLive bool

// StartWrapper starts the wrapper health check service
func StartWrapper() {
	// Start monitoring the main application's health
	go monitorMainAppHealth()
}

func monitorMainAppHealth() {
	for {
		resp, err := http.Get("http://localhost:8080/api/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			isMainAppLive = false
		} else {
			isMainAppLive = true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(5 * time.Second) // Check every 5 seconds
	}
} 
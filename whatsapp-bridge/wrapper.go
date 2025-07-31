package main

import (
	"fmt"
	"net/http"
	"time"
)

var isMainAppLive bool

func main() {
	// Start a lightweight health-check server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if isMainAppLive {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Main application is live."))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Main application is not live."))
		}
	})

	// Start monitoring the main application's health
	go monitorMainAppHealth()

	// Run the health-check server on a separate port
	fmt.Println("Starting wrapper health-check server on port 9091...")
	if err := http.ListenAndServe(":9091", nil); err != nil {
		fmt.Printf("Wrapper server error: %v\n", err)
	}
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
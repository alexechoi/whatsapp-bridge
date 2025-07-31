package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var isMainAppLive bool
var lastAlertSent time.Time

// AlertPayload represents the structure of the webhook alert
type AlertPayload struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	AppName   string    `json:"app_name"`
}

// StartWrapper starts the wrapper health check service
func StartWrapper() {
	// Start monitoring the main application's health
	go monitorMainAppHealth()
}

func monitorMainAppHealth() {
	var wasHealthy bool = true // Start assuming app is healthy
	
	for {
		resp, err := http.Get("http://localhost:8080/api/health")
		
		// Check if the app is healthy
		currentlyHealthy := err == nil && resp != nil && resp.StatusCode == http.StatusOK
		
		// Update global status
		isMainAppLive = currentlyHealthy
		
		// Close response body if it exists
		if resp != nil {
			resp.Body.Close()
		}
		
		// If app was healthy before but now isn't, send an alert
		if wasHealthy && !currentlyHealthy {
			fmt.Println("Health check failed: WhatsApp Bridge is unhealthy")
			sendWebhookAlert("unhealthy", "WhatsApp Bridge detected as unhealthy")
		} else if !wasHealthy && currentlyHealthy {
			// If app was unhealthy before but now is healthy, send recovery alert
			fmt.Println("Health check recovered: WhatsApp Bridge is now healthy")
			sendWebhookAlert("recovered", "WhatsApp Bridge has recovered and is now healthy")
		}
		
		// Update previous state
		wasHealthy = currentlyHealthy
		
		// Wait before next check
		time.Sleep(5 * time.Second)
	}
}

// sendWebhookAlert sends an alert to the configured webhook URL
func sendWebhookAlert(status, message string) {
	// Check if webhook URL is configured
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		// No webhook configured, skip
		return
	}
	
	// Prevent alert flooding - only send once per minute
	if time.Since(lastAlertSent) < time.Minute {
		return
	}
	
	// Create alert payload
	payload := AlertPayload{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		AppName:   "WhatsApp Bridge",
	}
	
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling webhook payload: %v\n", err)
		return
	}
	
	// Send HTTP POST request to webhook
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("Error sending webhook alert: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Webhook alert sent: %s (status: %s)\n", message, status)
	
	// Update last alert time
	lastAlertSent = time.Now()
} 
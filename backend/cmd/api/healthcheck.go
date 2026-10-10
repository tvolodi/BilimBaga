package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// healthURL is the API's own health route on loopback. The port is the one the server listens on: API_PORT, then
// BB_API_PORT, then 8080, the same order as config.Load (#473).
func healthURL() string {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("BB_API_PORT")
	}
	if port == "" {
		port = "8080"
	}
	return "http://127.0.0.1:" + port + "/api/v1/health"
}

// probeHealth returns nil only when the route answers 200 within the timeout (#473).
func probeHealth(url string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health answered %d", resp.StatusCode)
	}
	return nil
}

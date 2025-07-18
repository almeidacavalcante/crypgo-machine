package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthController handles health check endpoints
type HealthController struct{}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

// NewHealthController creates a new health controller
func NewHealthController() *HealthController {
	return &HealthController{}
}

var startTime = time.Now()

// Health handles the health check endpoint
func (c *HealthController) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:    "healthy",
		Service:   "crypgo-machine",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
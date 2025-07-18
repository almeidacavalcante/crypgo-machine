package api

import (
	"crypgo-machine/src/application/usecase"
	"encoding/json"
	"net/http"
	"strconv"
)

// TradingLogsController handles trading logs endpoints
type TradingLogsController struct {
	listTradingLogsUseCase *usecase.ListTradingLogsUseCase
}

// NewTradingLogsController creates a new trading logs controller
func NewTradingLogsController(listTradingLogsUseCase *usecase.ListTradingLogsUseCase) *TradingLogsController {
	return &TradingLogsController{
		listTradingLogsUseCase: listTradingLogsUseCase,
	}
}

// ListLogs handles GET /api/v1/trading/logs
func (c *TradingLogsController) ListLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	decision := r.URL.Query().Get("decision")
	limitStr := r.URL.Query().Get("limit")

	var limit int = 50 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Create input
	input := usecase.ListTradingLogsInput{
		Decision: decision,
		Limit:    limit,
	}

	// Execute use case
	output, err := c.listTradingLogsUseCase.Execute(input)
	if err != nil {
		http.Error(w, `{"error":"Failed to retrieve trading logs"}`, http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(output)
}
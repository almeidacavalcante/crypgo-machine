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
	symbol := r.URL.Query().Get("symbol")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var limit int = 20 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var offset int = 0 // default
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Create input
	input := usecase.ListTradingLogsInput{
		Decision: decision,
		Symbol:   symbol,
		Limit:    limit,
		Offset:   offset,
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
package controller

import (
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/scheduler"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SentimentController struct {
	generateUseCase   *usecase.GenerateSentimentSuggestionUseCase
	listUseCase       *usecase.ListSentimentSuggestionsUseCase
	approveUseCase    *usecase.ApproveSentimentSuggestionUseCase
	marketService     *service.MarketSentimentService
	sentimentScheduler *scheduler.SentimentScheduler
}

func NewSentimentController(
	generateUseCase *usecase.GenerateSentimentSuggestionUseCase,
	listUseCase *usecase.ListSentimentSuggestionsUseCase,
	approveUseCase *usecase.ApproveSentimentSuggestionUseCase,
	marketService *service.MarketSentimentService,
	sentimentScheduler *scheduler.SentimentScheduler,
) *SentimentController {
	return &SentimentController{
		generateUseCase:    generateUseCase,
		listUseCase:        listUseCase,
		approveUseCase:     approveUseCase,
		marketService:      marketService,
		sentimentScheduler: sentimentScheduler,
	}
}

// POST /api/v1/sentiment/generate
func (c *SentimentController) GenerateSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input usecase.GenerateSentimentSuggestionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body", err)
		return
	}

	output, err := c.generateUseCase.Execute(input)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input") {
			c.writeErrorResponse(w, http.StatusBadRequest, "Validation error", err)
		} else if strings.Contains(err.Error(), "pending suggestion") {
			c.writeErrorResponse(w, http.StatusConflict, "Pending suggestion exists", err)
		} else {
			c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate suggestion", err)
		}
		return
	}

	c.writeSuccessResponse(w, http.StatusCreated, "Sentiment suggestion generated successfully", output)
}

// GET /api/v1/sentiment/suggestions
func (c *SentimentController) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	input := usecase.ListSentimentSuggestionsInput{
		Status: query.Get("status"),
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			input.Limit = limit
		}
	}

	output, err := c.listUseCase.Execute(input)
	if err != nil {
		if strings.Contains(err.Error(), "invalid status") {
			c.writeErrorResponse(w, http.StatusBadRequest, "Invalid status parameter", err)
		} else {
			c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list suggestions", err)
		}
		return
	}

	c.writeSuccessResponse(w, http.StatusOK, "Suggestions retrieved successfully", output)
}

// POST /api/v1/sentiment/approve
func (c *SentimentController) ApproveSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input usecase.ApproveSentimentSuggestionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body", err)
		return
	}

	output, err := c.approveUseCase.Execute(input)
	if err != nil {
		if strings.Contains(err.Error(), "invalid input") || strings.Contains(err.Error(), "invalid action") {
			c.writeErrorResponse(w, http.StatusBadRequest, "Validation error", err)
		} else if strings.Contains(err.Error(), "not found") {
			c.writeErrorResponse(w, http.StatusNotFound, "Suggestion not found", err)
		} else if strings.Contains(err.Error(), "not pending") {
			c.writeErrorResponse(w, http.StatusConflict, "Suggestion already processed", err)
		} else {
			c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to process suggestion", err)
		}
		return
	}

	c.writeSuccessResponse(w, http.StatusOK, "Suggestion processed successfully", output)
}

// GET /api/v1/sentiment/analytics
func (c *SentimentController) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get analytics through list use case (with empty input to get just analytics)
	input := usecase.ListSentimentSuggestionsInput{Limit: 1}
	output, err := c.listUseCase.Execute(input)
	if err != nil {
		c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get analytics", err)
		return
	}

	// Return just the analytics part
	response := map[string]interface{}{
		"analytics": output.Analytics,
		"message":   "Analytics retrieved successfully",
	}

	c.writeSuccessResponse(w, http.StatusOK, "Analytics retrieved successfully", response)
}

// Helper methods for consistent response formatting
func (c *SentimentController) writeSuccessResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": true,
		"message": message,
		"data":    data,
	}

	json.NewEncoder(w).Encode(response)
}

func (c *SentimentController) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": false,
		"message": message,
		"error":   err.Error(),
	}

	json.NewEncoder(w).Encode(response)
}

// Health check for sentiment system
func (c *SentimentController) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "sentiment-analysis",
		"version":   "1.0.0",
		"endpoints": []string{
			"POST /api/v1/sentiment/generate",
			"GET /api/v1/sentiment/suggestions",
			"POST /api/v1/sentiment/approve",
			"GET /api/v1/sentiment/analytics",
		},
		"message": "Sentiment analysis service is operational",
	}

	c.writeSuccessResponse(w, http.StatusOK, "Service healthy", health)
}

// POST /api/v1/sentiment/analyze
func (c *SentimentController) TriggerManualAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if market service is available
	if c.marketService == nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Market sentiment service not available", 
			fmt.Errorf("service not initialized"))
		return
	}

	// Perform manual sentiment analysis
	result, err := c.marketService.CollectMarketSentiment()
	if err != nil {
		c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to perform sentiment analysis", err)
		return
	}

	// Format response
	response := map[string]interface{}{
		"suggestion_id": result.Suggestion.GetId().GetValue(),
		"sentiment":     result.Suggestion.GetLevel(),
		"score":         result.Suggestion.GetOverallScore(),
		"confidence":    result.Confidence,
		"reasoning":     result.Reasoning,
		"sources": map[string]interface{}{
			"fear_greed_index": result.Sources.GetFearGreedIndex(),
			"news_score":       result.Sources.GetNewsScore(),
			"reddit_score":     result.Sources.GetRedditScore(),
			"social_score":     result.Sources.GetSocialScore(),
		},
		"suggestions": c.marketService.GetSentimentSuggestions(result.Suggestion.GetLevel()),
		"timestamp":   time.Now(),
	}

	c.writeSuccessResponse(w, http.StatusOK, "Manual sentiment analysis completed", response)
}

// POST /api/v1/sentiment/quick-check
func (c *SentimentController) QuickSentimentCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.marketService == nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Market sentiment service not available", 
			fmt.Errorf("service not initialized"))
		return
	}

	result, err := c.marketService.QuickSentimentCheck()
	if err != nil {
		c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to perform quick sentiment check", err)
		return
	}

	response := map[string]interface{}{
		"sentiment":  result.Suggestion.GetLevel(),
		"score":      result.Suggestion.GetOverallScore(),
		"confidence": result.Confidence,
		"reasoning":  result.Reasoning,
		"timestamp":  time.Now(),
		"type":       "quick_check",
	}

	c.writeSuccessResponse(w, http.StatusOK, "Quick sentiment check completed", response)
}

// GET /api/v1/sentiment/scheduler/status
func (c *SentimentController) GetSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.sentimentScheduler == nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Sentiment scheduler not available", 
			fmt.Errorf("scheduler not initialized"))
		return
	}

	status := map[string]interface{}{
		"running":   c.sentimentScheduler.IsRunning(),
		"timestamp": time.Now(),
	}

	c.writeSuccessResponse(w, http.StatusOK, "Scheduler status retrieved", status)
}

// POST /api/v1/sentiment/scheduler/trigger
func (c *SentimentController) TriggerScheduledAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.sentimentScheduler == nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Sentiment scheduler not available", 
			fmt.Errorf("scheduler not initialized"))
		return
	}

	err := c.sentimentScheduler.TriggerManualAnalysis()
	if err != nil {
		c.writeErrorResponse(w, http.StatusInternalServerError, "Failed to trigger scheduled analysis", err)
		return
	}

	response := map[string]interface{}{
		"message":   "Scheduled analysis triggered successfully",
		"timestamp": time.Now(),
	}

	c.writeSuccessResponse(w, http.StatusAccepted, "Analysis triggered", response)
}

// GET /api/v1/sentiment/data-sources/validate
func (c *SentimentController) ValidateDataSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if c.marketService == nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Market sentiment service not available", 
			fmt.Errorf("service not initialized"))
		return
	}

	err := c.marketService.ValidateDataSources()
	if err != nil {
		c.writeErrorResponse(w, http.StatusServiceUnavailable, "Data source validation failed", err)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"message":   "All data sources are accessible",
		"timestamp": time.Now(),
		"sources": []string{
			"Fear & Greed Index API",
			"RSS Feeds (CoinDesk, CoinTelegraph, Reddit)",
		},
	}

	c.writeSuccessResponse(w, http.StatusOK, "Data sources validated", response)
}
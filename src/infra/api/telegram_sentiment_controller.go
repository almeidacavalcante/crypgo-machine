package api

import (
	"crypgo-machine/src/infra/notification"
	"encoding/json"
	"net/http"
)

type TelegramSentimentController struct {
	sentimentConsumer *notification.TelegramSentimentConsumer
}

func NewTelegramSentimentController(sentimentConsumer *notification.TelegramSentimentConsumer) *TelegramSentimentController {
	return &TelegramSentimentController{
		sentimentConsumer: sentimentConsumer,
	}
}

// POST /api/v1/telegram/test-sentiment
func (t *TelegramSentimentController) TestSentimentNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if t.sentimentConsumer == nil {
		t.writeErrorResponse(w, http.StatusServiceUnavailable, "Sentiment consumer not available")
		return
	}

	err := t.sentimentConsumer.SendTestSentimentNotification()
	if err != nil {
		t.writeErrorResponse(w, http.StatusInternalServerError, "Failed to send test sentiment notification: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Test sentiment notification sent successfully",
		"type":    "sentiment_test",
	}

	t.writeSuccessResponse(w, http.StatusOK, response)
}

// GET /api/v1/telegram/sentiment/status
func (t *TelegramSentimentController) GetSentimentNotificationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"sentiment_consumer_available": t.sentimentConsumer != nil,
		"service":                      "telegram-sentiment-notifications",
		"endpoints": []string{
			"POST /api/v1/telegram/test-sentiment",
			"GET /api/v1/telegram/sentiment/status",
		},
	}

	t.writeSuccessResponse(w, http.StatusOK, status)
}

func (t *TelegramSentimentController) writeSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	json.NewEncoder(w).Encode(response)
}

func (t *TelegramSentimentController) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}

	json.NewEncoder(w).Encode(response)
}
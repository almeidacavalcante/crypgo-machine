package notification

import (
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/infra/queue"
	"crypgo-machine/src/infra/scheduler"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

type TelegramSentimentConsumer struct {
	broker          queue.MessageBroker
	exchangeName    string
	queueName       string
	telegramService *TelegramService
}

func NewTelegramSentimentConsumer(
	broker queue.MessageBroker,
	exchangeName string,
	queueName string,
	telegramService *TelegramService,
) *TelegramSentimentConsumer {
	return &TelegramSentimentConsumer{
		broker:          broker,
		exchangeName:    exchangeName,
		queueName:       queueName,
		telegramService: telegramService,
	}
}

func (t *TelegramSentimentConsumer) Start() error {
	if !t.telegramService.IsEnabled() {
		log.Println("⚠️ Telegram service not enabled, skipping sentiment consumer start")
		return nil
	}

	routingKeys := []string{
		"sentiment.analysis.completed",
		"sentiment.suggestion.pending",
		"sentiment.extreme.detected",
	}

	return t.broker.Subscribe(t.exchangeName, t.queueName, routingKeys, t.handleMessage)
}

func (t *TelegramSentimentConsumer) handleMessage(msg queue.Message) error {
	if !t.telegramService.IsEnabled() {
		log.Println("⚠️ Telegram service not enabled, skipping sentiment message")
		return nil
	}

	switch msg.RoutingKey {
	case "sentiment.analysis.completed":
		return t.handleSentimentAnalysisCompleted(msg)
	case "sentiment.suggestion.pending":
		return t.handleSentimentSuggestionPending(msg)
	case "sentiment.extreme.detected":
		return t.handleExtremeSentimentDetected(msg)
	default:
		log.Printf("Unknown sentiment routing key: %s", msg.RoutingKey)
		return nil
	}
}

func (t *TelegramSentimentConsumer) handleSentimentAnalysisCompleted(msg queue.Message) error {
	var payload scheduler.SentimentNotificationPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal sentiment notification: %w", err)
	}

	message := t.formatSentimentAnalysisMessage(payload)
	return t.telegramService.SendSimpleMessage(message)
}

func (t *TelegramSentimentConsumer) handleSentimentSuggestionPending(msg queue.Message) error {
	var payload scheduler.SentimentNotificationPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal sentiment suggestion: %w", err)
	}

	message := t.formatSentimentSuggestionMessage(payload)
	return t.telegramService.SendSimpleMessage(message)
}

func (t *TelegramSentimentConsumer) handleExtremeSentimentDetected(msg queue.Message) error {
	var payload scheduler.SentimentNotificationPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal extreme sentiment alert: %w", err)
	}

	message := t.formatExtremeSentimentMessage(payload)
	return t.telegramService.SendSimpleMessage(message)
}

func (t *TelegramSentimentConsumer) formatSentimentAnalysisMessage(payload scheduler.SentimentNotificationPayload) string {
	// Get sentiment emoji
	sentimentEmoji := t.getSentimentEmoji(payload.Sentiment)
	analysisTypeEmoji := "📊"
	if payload.Type == "quick_check" {
		analysisTypeEmoji = "⚡"
	}

	// Format score percentage
	scorePercent := payload.Score * 100
	confidencePercent := payload.Confidence * 100

	// Format timestamp
	timeStr := payload.Timestamp.Format("15:04")

	message := fmt.Sprintf(
		"%s <b>Market Sentiment Analysis</b> - %s\n\n"+
			"%s <b>Overall Sentiment</b>: %s (%.1f%%)\n"+
			"🎯 <b>Confidence</b>: %.1f%%\n"+
			"⏰ <b>Time</b>: %s\n\n"+
			"💡 <b>Reasoning</b>:\n%s\n\n"+
			"📈 <b>SUGESTÕES CONSULTIVAS</b>:\n"+
			"• Trade Amount: <code>%.1fx</code> multiplier\n"+
			"• Profit Target: <code>%.1f%%</code>\n"+
			"• Interval: <code>%s</code>\n"+
			"• Action: <code>%s</code>\n\n"+
			"⚠️ <i>Estas são apenas sugestões. Revise e aprove manualmente antes de aplicar.</i>\n\n"+
			"#CrypGo #SentimentAnalysis #%s",
		analysisTypeEmoji,
		timeStr,
		sentimentEmoji,
		t.getSentimentDisplayName(payload.Sentiment),
		scorePercent,
		confidencePercent,
		timeStr,
		payload.Reasoning,
		payload.Suggestions.TradeAmountMultiplier,
		payload.Suggestions.MinimumProfitThreshold,
		t.formatInterval(payload.Suggestions.IntervalSeconds),
		payload.Suggestions.Recommendation,
		payload.Type,
	)

	return message
}

func (t *TelegramSentimentConsumer) formatSentimentSuggestionMessage(payload scheduler.SentimentNotificationPayload) string {
	sentimentEmoji := t.getSentimentEmoji(payload.Sentiment)
	
	message := fmt.Sprintf(
		"🔔 <b>Nova Sugestão de Sentiment</b>\n\n"+
			"%s <b>Sentiment</b>: %s\n"+
			"📊 <b>Score</b>: %.3f\n"+
			"🎯 <b>Confidence</b>: %.1f%%\n"+
			"🆔 <b>Suggestion ID</b>: <code>%s</code>\n\n"+
			"💭 <b>Reasoning</b>:\n%s\n\n"+
			"❓ <b>Ação Necessária</b>:\n"+
			"Acesse o dashboard para revisar e aprovar esta sugestão.\n\n"+
			"#CrypGo #PendingSuggestion #ApprovalRequired",
		sentimentEmoji,
		t.getSentimentDisplayName(payload.Sentiment),
		payload.Score,
		payload.Confidence*100,
		payload.SuggestionID,
		payload.Reasoning,
	)

	return message
}

func (t *TelegramSentimentConsumer) formatExtremeSentimentMessage(payload scheduler.SentimentNotificationPayload) string {
	var alertEmoji string
	var alertType string

	if payload.Sentiment == "very_bullish" {
		alertEmoji = "🚀"
		alertType = "EXTREME BULLISH"
	} else if payload.Sentiment == "very_bearish" {
		alertEmoji = "🔴"
		alertType = "EXTREME BEARISH"
	} else {
		alertEmoji = "⚠️"
		alertType = "EXTREME SENTIMENT"
	}

	message := fmt.Sprintf(
		"%s <b>ALERTA: %s DETECTADO</b> %s\n\n"+
			"📊 <b>Sentiment</b>: %s\n"+
			"📈 <b>Score</b>: %.3f\n"+
			"🎯 <b>Confidence</b>: %.1f%%\n"+
			"⏰ <b>Detected at</b>: %s\n\n"+
			"💭 <b>Analysis</b>:\n%s\n\n"+
			"⚡ <b>AÇÃO RECOMENDADA</b>:\n"+
			"Review your trading strategy immediately. Consider %s.\n\n"+
			"#CrypGo #ExtremeAlert #%s",
		alertEmoji,
		alertType,
		alertEmoji,
		t.getSentimentDisplayName(payload.Sentiment),
		payload.Score,
		payload.Confidence*100,
		payload.Timestamp.Format("15:04:05"),
		payload.Reasoning,
		payload.Suggestions.ReasoningText,
		payload.Sentiment,
	)

	return message
}

func (t *TelegramSentimentConsumer) getSentimentEmoji(sentiment string) string {
	switch sentiment {
	case "very_bullish":
		return "🚀"
	case "bullish":
		return "📈"
	case "neutral":
		return "➡️"
	case "bearish":
		return "📉"
	case "very_bearish":
		return "🔴"
	default:
		return "❓"
	}
}

func (t *TelegramSentimentConsumer) getSentimentDisplayName(sentiment string) string {
	switch sentiment {
	case "very_bullish":
		return "Very Bullish"
	case "bullish":
		return "Bullish"
	case "neutral":
		return "Neutral"
	case "bearish":
		return "Bearish"
	case "very_bearish":
		return "Very Bearish"
	default:
		return "Unknown"
	}
}

func (t *TelegramSentimentConsumer) formatInterval(seconds int) string {
	switch seconds {
	case 300:
		return "5 min"
	case 600:
		return "10 min"
	case 900:
		return "15 min"
	case 1800:
		return "30 min"
	case 3600:
		return "1 hour"
	default:
		minutes := seconds / 60
		if minutes < 60 {
			return strconv.Itoa(minutes) + " min"
		}
		hours := minutes / 60
		return strconv.Itoa(hours) + " hour(s)"
	}
}

// SendTestSentimentNotification sends a test sentiment notification
func (t *TelegramSentimentConsumer) SendTestSentimentNotification() error {
	if !t.telegramService.IsEnabled() {
		return fmt.Errorf("telegram service not enabled")
	}

	testPayload := scheduler.SentimentNotificationPayload{
		SuggestionID: "test-suggestion-123",
		Sentiment:    "bullish",
		Score:        0.25,
		Confidence:   0.75,
		Reasoning:    "Test sentiment analysis with positive market indicators from Fear & Greed Index and recent news coverage.",
		Suggestions: service.SentimentTradingSuggestions{
			TradeAmountMultiplier:  1.2,
			MinimumProfitThreshold: 1.0,
			IntervalSeconds:        600,
			Recommendation:         "normal_plus",
			ReasoningText:          "Sentiment positivo - ligeiro aumento na agressividade",
		},
		Timestamp: time.Now(),
		Type:      "test_analysis",
	}

	message := t.formatSentimentAnalysisMessage(testPayload)
	return t.telegramService.SendSimpleMessage(message)
}
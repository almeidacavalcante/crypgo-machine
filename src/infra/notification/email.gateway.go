package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"crypgo-machine/src/infra/queue"
)

type EmailNotificationConsumer struct {
	broker       queue.MessageBroker
	exchangeName string
	queueName    string
	emailService *EmailService
	targetEmail  string
}

func NewEmailNotificationConsumer(
	broker queue.MessageBroker,
	exchangeName string,
	queueName string,
	emailService *EmailService,
	targetEmail string,
) *EmailNotificationConsumer {
	return &EmailNotificationConsumer{
		broker:       broker,
		exchangeName: exchangeName,
		queueName:    queueName,
		emailService: emailService,
		targetEmail:  targetEmail,
	}
}

func (e *EmailNotificationConsumer) Start() error {
	routingKeys := []string{
		"trading_bot.created",
		"trading_bot.started",
		"trading_bot.stopped",
		"trading.buy_executed",
		"trading.sell_executed",
	}

	return e.broker.Subscribe(e.exchangeName, e.queueName, routingKeys, e.handleMessage)
}

func (e *EmailNotificationConsumer) handleMessage(msg queue.Message) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("❌ Error unmarshaling message payload: %v", err)
		return err
	}

	switch msg.RoutingKey {
	case "trading_bot.created":
		subject := "🤖 CrypGo: Trading Bot Criado"
		body := fmt.Sprintf("O bot %v foi criado com sucesso para o símbolo %v.", payload["id"], payload["symbol"])
		return e.sendSimpleEmail(subject, body)

	case "trading_bot.started":
		subject := "▶️ CrypGo: Trading Bot Iniciado"
		body := fmt.Sprintf("O bot %v está agora rodando para %v.", payload["id"], payload["symbol"])
		return e.sendSimpleEmail(subject, body)

	case "trading_bot.stopped":
		subject := "⏹️ CrypGo: Trading Bot Pausado"
		body := fmt.Sprintf("O bot %v foi interrompido.", payload["id"])
		return e.sendSimpleEmail(subject, body)

	case "trading.buy_executed":
		return e.handleTradingEvent(payload, true)

	case "trading.sell_executed":
		return e.handleTradingEvent(payload, false)

	default:
		log.Println("Evento ignorado:", msg.RoutingKey)
		return nil
	}
}

func (e *EmailNotificationConsumer) sendSimpleEmail(subject, body string) error {
	emailData := EmailData{
		To:      e.targetEmail,
		Subject: subject,
		Body:    body,
	}
	return e.emailService.SendEmail(emailData)
}

func (e *EmailNotificationConsumer) handleTradingEvent(payload map[string]interface{}, isBuy bool) error {
	// Convert payload to TradingEventData
	tradingData := e.payloadToTradingEventData(payload)

	var subject, body string
	if isBuy {
		subject, body = GenerateBuyEmailTemplate(tradingData)
	} else {
		subject, body = GenerateSellEmailTemplate(tradingData)
	}

	emailData := EmailData{
		To:      e.targetEmail,
		Subject: subject,
		Body:    body,
	}

	return e.emailService.SendEmail(emailData)
}

func (e *EmailNotificationConsumer) payloadToTradingEventData(payload map[string]interface{}) TradingEventData {
	data := TradingEventData{
		BotID:       getStringValue(payload, "bot_id"),
		Symbol:      getStringValue(payload, "symbol"),
		Action:      getStringValue(payload, "action"),
		Price:       getFloatValue(payload, "price"),
		Quantity:    getFloatValue(payload, "quantity"),
		TotalValue:  getFloatValue(payload, "total_value"),
		Strategy:    getStringValue(payload, "strategy"),
		TradingFees: getFloatValue(payload, "trading_fees"),
		Currency:    getStringValue(payload, "currency"),
		Timestamp:   time.Now(), // Default fallback
	}

	// Parse timestamp if available
	if timestampStr, ok := payload["timestamp"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			data.Timestamp = parsedTime
		}
	}

	// Add sell-specific fields
	if action := getStringValue(payload, "action"); action == "sell_executed" {
		data.EntryPrice = getFloatValue(payload, "entry_price")
		data.ProfitLoss = getFloatValue(payload, "profit_loss")
		data.ProfitLossPerc = getFloatValue(payload, "profit_loss_perc")
	}

	return data
}

func getStringValue(payload map[string]interface{}, key string) string {
	if val, ok := payload[key].(string); ok {
		return val
	}
	return ""
}

func getFloatValue(payload map[string]interface{}, key string) float64 {
	if val, ok := payload[key].(float64); ok {
		return val
	}
	if val, ok := payload[key].(int); ok {
		return float64(val)
	}
	return 0.0
}

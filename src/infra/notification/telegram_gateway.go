package notification

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"crypgo-machine/src/infra/queue"
)

type TelegramNotificationConsumer struct {
	broker          queue.MessageBroker
	exchangeName    string
	queueName       string
	telegramService *TelegramService
}

func NewTelegramNotificationConsumer(
	broker queue.MessageBroker,
	exchangeName string,
	queueName string,
	telegramService *TelegramService,
) *TelegramNotificationConsumer {
	return &TelegramNotificationConsumer{
		broker:          broker,
		exchangeName:    exchangeName,
		queueName:       queueName,
		telegramService: telegramService,
	}
}

func (t *TelegramNotificationConsumer) Start() error {
	if !t.telegramService.IsEnabled() {
		log.Println("⚠️ Telegram service not enabled, skipping consumer start")
		return nil
	}
	
	routingKeys := []string{
		"trading_bot.created",
		"trading_bot.started",
		"trading_bot.stopped",
		"trading.buy_executed",
		"trading.sell_executed",
	}

	return t.broker.Subscribe(t.exchangeName, t.queueName, routingKeys, t.handleMessage)
}

func (t *TelegramNotificationConsumer) handleMessage(msg queue.Message) error {
	if !t.telegramService.IsEnabled() {
		log.Println("⚠️ Telegram service not enabled, skipping message")
		return nil
	}
	
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		log.Printf("❌ Error unmarshaling message payload: %v", err)
		return err
	}

	switch msg.RoutingKey {
	case "trading_bot.created":
		message := fmt.Sprintf("🤖 <b>CrypGo: Trading Bot Criado</b>\n\nBot %v criado com sucesso para <b>%v</b>", 
			payload["id"], payload["symbol"])
		return t.sendSimpleMessage(message)

	case "trading_bot.started":
		message := fmt.Sprintf("▶️ <b>CrypGo: Trading Bot Iniciado</b>\n\nBot %v está rodando para <b>%v</b>", 
			payload["id"], payload["symbol"])
		return t.sendSimpleMessage(message)

	case "trading_bot.stopped":
		message := fmt.Sprintf("⏹️ <b>CrypGo: Trading Bot Pausado</b>\n\nBot %v foi interrompido", 
			payload["id"])
		return t.sendSimpleMessage(message)

	case "trading.buy_executed":
		return t.handleTradingEvent(payload, true)

	case "trading.sell_executed":
		return t.handleTradingEvent(payload, false)

	default:
		log.Println("Evento ignorado no Telegram:", msg.RoutingKey)
		return nil
	}
}

func (t *TelegramNotificationConsumer) sendSimpleMessage(message string) error {
	return t.telegramService.SendSimpleMessage(message)
}

func (t *TelegramNotificationConsumer) handleTradingEvent(payload map[string]interface{}, isBuy bool) error {
	// Convert payload to structured data
	tradingData := t.payloadToTradingEventData(payload)

	var message string
	if isBuy {
		message = t.generateBuyMessage(tradingData)
	} else {
		message = t.generateSellMessage(tradingData)
	}

	return t.sendSimpleMessage(message)
}

func (t *TelegramNotificationConsumer) generateBuyMessage(data TradingEventData) string {
	return fmt.Sprintf(
		"💰 <b>COMPRA EXECUTADA</b>\n\n"+
			"🤖 Bot: <code>%s</code>\n"+
			"💱 Par: <b>%s</b>\n"+
			"💵 Preço: <b>%.8f %s</b>\n"+
			"📊 Quantidade: <b>%.8f</b>\n"+
			"💸 Total: <b>%.2f %s</b>\n"+
			"🎯 Estratégia: <code>%s</code>\n"+
			"⏰ %s",
		data.BotID,
		data.Symbol,
		data.Price, data.Currency,
		data.Quantity,
		data.TotalValue, data.Currency,
		data.Strategy,
		data.Timestamp.Format("15:04:05"),
	)
}

func (t *TelegramNotificationConsumer) generateSellMessage(data TradingEventData) string {
	profitEmoji := "📈"
	if data.ProfitLoss < 0 {
		profitEmoji = "📉"
	}

	return fmt.Sprintf(
		"💸 <b>VENDA EXECUTADA</b>\n\n"+
			"🤖 Bot: <code>%s</code>\n"+
			"💱 Par: <b>%s</b>\n"+
			"💵 Preço Venda: <b>%.8f %s</b>\n"+
			"💰 Preço Compra: <b>%.8f %s</b>\n"+
			"📊 Quantidade: <b>%.8f</b>\n"+
			"💸 Total: <b>%.2f %s</b>\n"+
			"%s P&L: <b>%.2f %s (%.2f%%)</b>\n"+
			"🎯 Estratégia: <code>%s</code>\n"+
			"⏰ %s",
		data.BotID,
		data.Symbol,
		data.Price, data.Currency,
		data.EntryPrice, data.Currency,
		data.Quantity,
		data.TotalValue, data.Currency,
		profitEmoji, data.ProfitLoss, data.Currency, data.ProfitLossPerc,
		data.Strategy,
		data.Timestamp.Format("15:04:05"),
	)
}

func (t *TelegramNotificationConsumer) payloadToTradingEventData(payload map[string]interface{}) TradingEventData {
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
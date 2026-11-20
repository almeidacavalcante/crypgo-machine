package usecase

import (
	"encoding/json"
	"testing"

	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"

	"github.com/adshao/go-binance/v2"
)

// The "trading_bot.created" event must carry the bot id and symbol as plain
// strings: the email and Telegram consumers print payload["id"] and
// payload["symbol"] directly.
func TestCreateTradingBotUseCase_CreatedEventCarriesIdAndSymbolAsStrings(t *testing.T) {
	var savedBot *entity.TradingBot
	repo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradingBot) error {
			savedBot = bot
			return nil
		},
	}
	broker := &recordingLifecycleBroker{}
	uc := NewCreateTradingBotUseCase(repo, binance.Client{}, broker, "test-exchange")

	err := uc.Execute(InputCreateTradingBot{
		Symbol:                 "SOLBRL",
		Quantity:               1.5,
		Strategy:               "MovingAverage",
		Params:                 service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:        3600,
		InitialCapital:         10000.0,
		TradeAmount:            4000.0,
		Currency:               "BRL",
		TradingFees:            0.1,
		MinimumProfitThreshold: 5.0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if savedBot == nil {
		t.Fatal("expected the bot to be saved")
	}

	messages, exchanges := broker.messagesWithKey("trading_bot.created")
	if len(messages) != 1 {
		t.Fatalf("expected 1 trading_bot.created event, got %d", len(messages))
	}
	if exchanges[0] != "test-exchange" {
		t.Errorf("expected exchange test-exchange, got %s", exchanges[0])
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(messages[0].Payload, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	id, ok := payload["id"].(string)
	if !ok {
		t.Fatalf("expected id to be a string, got %T (%v)", payload["id"], payload["id"])
	}
	if id != savedBot.Id.GetValue() {
		t.Errorf("expected id %s, got %s", savedBot.Id.GetValue(), id)
	}

	symbol, ok := payload["symbol"].(string)
	if !ok {
		t.Fatalf("expected symbol to be a string, got %T (%v)", payload["symbol"], payload["symbol"])
	}
	if symbol != "SOLBRL" {
		t.Errorf("expected symbol SOLBRL, got %s", symbol)
	}
}

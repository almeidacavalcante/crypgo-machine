package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
	"github.com/adshao/go-binance/v2"
	"math"
	"strconv"
	"testing"
	"time"
)

// MockTradeBotRepository implements repository.TradingBotRepository for testing
type MockTradeBotRepository struct{}

func (m *MockTradeBotRepository) Save(bot *entity.TradingBot) error                  { return nil }
func (m *MockTradeBotRepository) Update(bot *entity.TradingBot) error                { return nil }
func (m *MockTradeBotRepository) Exists(id string) (bool, error)                     { return false, nil }
func (m *MockTradeBotRepository) GetTradeByID(id string) (*entity.TradingBot, error) { return nil, nil }
func (m *MockTradeBotRepository) GetAllTradingBots() ([]*entity.TradingBot, error)   { return nil, nil }
func (m *MockTradeBotRepository) GetTradingBotsByStatus(status entity.Status) ([]*entity.TradingBot, error) {
	return nil, nil
}

// MockTradingDecisionLogRepository implements repository.TradingDecisionLogRepository for testing
type MockTradingDecisionLogRepository struct{}

func (m *MockTradingDecisionLogRepository) Save(log *entity.TradingDecisionLog) error { return nil }
func (m *MockTradingDecisionLogRepository) GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error) {
	return nil, nil
}
func (m *MockTradingDecisionLogRepository) GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error) {
	return nil, nil
}
func (m *MockTradingDecisionLogRepository) GetRecentLogs(limit int) ([]*entity.TradingDecisionLog, error) {
	return nil, nil
}
func (m *MockTradingDecisionLogRepository) GetRecentLogsByDecision(decision string, limit int) ([]*entity.TradingDecisionLog, error) {
	return nil, nil
}
func (m *MockTradingDecisionLogRepository) GetLogsWithFilters(decision string, symbol string, limit int, offset int) ([]*entity.TradingDecisionLog, int, error) {
	return nil, 0, nil
}

// MockMessageBroker implements queue.MessageBroker for testing
type MockMessageBroker struct{}

func (m *MockMessageBroker) Publish(exchangeName string, message queue.Message) error { return nil }
func (m *MockMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	return nil
}
func (m *MockMessageBroker) Close() error { return nil }

func TestLiveTradingExecutionContext_FeeCalculation(t *testing.T) {
	// Setup
	repo := &MockTradeBotRepository{}
	client := external.NewBinanceClientFake()
	broker := &MockMessageBroker{}
	decisionRepo := &MockTradingDecisionLogRepository{}

	ctx := NewLiveTradingExecutionContext(client, repo, decisionRepo, broker, "test_exchange")

	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := entity.NewMovingAverageStrategy(5, 20)

	t.Run("Fixed quantity trading with fees", func(t *testing.T) {
		// Create bot with fixed quantity (true)
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 100, "BRL", 0.1, 2.0, true)

		currentPrice := 300000.0
		timestamp := time.Now()

		// Execute buy order
		err := ctx.ExecuteTrade(entity.Buy, bot, currentPrice, timestamp)
		if err != nil {
			t.Fatalf("Buy order failed: %v", err)
		}

		// Verify actual quantity held after fees (0.001 * (1 - 0.001) = 0.0009999)
		expectedActualQuantity := 0.001 * (1.0 - 0.1/100.0)
		actualQuantity := bot.GetActualQuantityHeld()

		if actualQuantity != expectedActualQuantity {
			t.Errorf("Expected actual quantity %.8f, got %.8f", expectedActualQuantity, actualQuantity)
		}

		// Execute sell order
		err = ctx.ExecuteTrade(entity.Sell, bot, currentPrice, timestamp)
		if err != nil {
			t.Fatalf("Sell order failed: %v", err)
		}

		// Verify quantities are cleared after sell
		if bot.GetActualQuantityHeld() != 0 {
			t.Errorf("Expected actual quantity to be cleared after sell, got %.8f", bot.GetActualQuantityHeld())
		}
	})

	t.Run("Dynamic quantity trading with fees", func(t *testing.T) {
		// Create bot with dynamic quantity (false)
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 300, "BRL", 0.1, 2.0, false)

		currentPrice := 300000.0
		timestamp := time.Now()

		// Execute buy order - should calculate quantity from trade amount
		err := ctx.ExecuteTrade(entity.Buy, bot, currentPrice, timestamp)
		if err != nil {
			t.Fatalf("Buy order failed: %v", err)
		}

		// Expected quantity: 300 BRL / 300000 BRL/BTC = 0.001 BTC
		expectedQuantity := 300.0 / currentPrice
		expectedActualQuantity := expectedQuantity * (1.0 - 0.1/100.0)
		actualQuantity := bot.GetActualQuantityHeld()

		if actualQuantity != expectedActualQuantity {
			t.Errorf("Expected actual quantity %.8f, got %.8f", expectedActualQuantity, actualQuantity)
		}
	})

	t.Run("CalculateQuantityForSell uses actual quantity", func(t *testing.T) {
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 100, "BRL", 0.1, 2.0, true)

		// Simulate buy with actual quantity after fees
		bot.SetActualQuantityHeld(0.0009999)

		sellQuantity := bot.CalculateQuantityForSell()

		if sellQuantity != 0.0009999 {
			t.Errorf("Expected sell quantity %.8f, got %.8f", 0.0009999, sellQuantity)
		}
	})

	t.Run("CalculateQuantityForSell fallback to estimated fees", func(t *testing.T) {
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 100, "BRL", 0.1, 2.0, true)

		// No actual quantity set (actualQuantityHeld = 0)
		sellQuantity := bot.CalculateQuantityForSell()
		expectedQuantity := 0.001 * (1.0 - 0.1/100.0)

		if sellQuantity != expectedQuantity {
			t.Errorf("Expected sell quantity %.8f, got %.8f", expectedQuantity, sellQuantity)
		}
	})
}

// recordingMessageBroker captures published messages for assertions
type recordingMessageBroker struct {
	messages []queue.Message
}

func (m *recordingMessageBroker) Publish(exchangeName string, message queue.Message) error {
	m.messages = append(m.messages, message)
	return nil
}
func (m *recordingMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	return nil
}
func (m *recordingMessageBroker) Close() error { return nil }

func (m *recordingMessageBroker) lastQuantity(t *testing.T, routingKey string) float64 {
	t.Helper()
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].RoutingKey != routingKey {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(m.messages[i].Payload, &payload); err != nil {
			t.Fatalf("Failed to decode %s payload: %v", routingKey, err)
		}
		quantity, ok := payload["quantity"].(float64)
		if !ok {
			t.Fatalf("Event %s has no numeric quantity: %v", routingKey, payload["quantity"])
		}
		return quantity
	}
	t.Fatalf("No %s event was published", routingKey)
	return 0
}

func TestLiveTradingExecutionContext_RecordsQuantitySentAfterStepSizeAdjustment(t *testing.T) {
	symbol, _ := vo.NewSymbol("SOLBRL")
	strategy := entity.NewMovingAverageStrategy(5, 20)
	timestamp := time.Now()
	const tradingFees = 0.1
	feeFactor := 1.0 - tradingFees/100.0

	t.Run("Dynamic quantity not aligned to step size", func(t *testing.T) {
		client := external.NewBinanceClientFake()
		broker := &recordingMessageBroker{}
		ctx := NewLiveTradingExecutionContext(client, &MockTradeBotRepository{}, &MockTradingDecisionLogRepository{}, broker, "test_exchange")

		// 100 BRL / 808.90 = 0.123625... SOL; SOLBRL step size is 0.01, so 0.12 is sent
		bot := entity.NewTradingBot(symbol, 0.01, strategy, 60, 1000, 100, "BRL", tradingFees, 2.0, false)
		currentPrice := 808.90

		if err := ctx.ExecuteTrade(entity.Buy, bot, currentPrice, timestamp); err != nil {
			t.Fatalf("Buy order failed: %v", err)
		}
		if !bot.GetIsPositioned() {
			t.Fatal("Expected bot to be positioned after buy")
		}

		if sent := client.GetLastOrderQuantity(); sent != "0.12" {
			t.Fatalf("Expected quantity sent to Binance to be 0.12, got %q", sent)
		}

		sentQuantity := 0.12
		held := bot.GetActualQuantityHeld()
		if held > sentQuantity {
			t.Errorf("Recorded held quantity %.8f is larger than the quantity sent %.8f", held, sentQuantity)
		}
		expectedHeld := sentQuantity * feeFactor
		if math.Abs(held-expectedHeld) > 1e-12 {
			t.Errorf("Expected held quantity %.8f, got %.8f", expectedHeld, held)
		}

		if got := broker.lastQuantity(t, "trading.buy_executed"); math.Abs(got-sentQuantity) > 1e-12 {
			t.Errorf("Expected buy event quantity %.8f, got %.8f", sentQuantity, got)
		}

		// The following sell must be accepted and close the position
		if err := ctx.ExecuteTrade(entity.Sell, bot, currentPrice, timestamp); err != nil {
			t.Fatalf("Sell order failed: %v", err)
		}
		if bot.GetIsPositioned() {
			t.Fatal("Expected bot to be out of position after sell")
		}
		soldQuantity, err := strconv.ParseFloat(client.GetLastOrderQuantity(), 64)
		if err != nil {
			t.Fatalf("Failed to parse sell quantity: %v", err)
		}
		if soldQuantity > held {
			t.Errorf("Sell quantity %.8f is larger than the held quantity %.8f", soldQuantity, held)
		}
		if got := broker.lastQuantity(t, "trading.sell_executed"); math.Abs(got-soldQuantity) > 1e-12 {
			t.Errorf("Expected sell event quantity %.8f, got %.8f", soldQuantity, got)
		}
	})

	t.Run("Executed quantity from the exchange takes precedence", func(t *testing.T) {
		client := external.NewBinanceClientFake()
		client.SetExecutedQuantityOverride("0.05")
		broker := &recordingMessageBroker{}
		ctx := NewLiveTradingExecutionContext(client, &MockTradeBotRepository{}, &MockTradingDecisionLogRepository{}, broker, "test_exchange")

		bot := entity.NewTradingBot(symbol, 0.01, strategy, 60, 1000, 100, "BRL", tradingFees, 2.0, false)

		if err := ctx.ExecuteTrade(entity.Buy, bot, 808.90, timestamp); err != nil {
			t.Fatalf("Buy order failed: %v", err)
		}

		expectedHeld := 0.05 * feeFactor
		if held := bot.GetActualQuantityHeld(); math.Abs(held-expectedHeld) > 1e-12 {
			t.Errorf("Expected held quantity %.8f from executed quantity, got %.8f", expectedHeld, held)
		}
		if got := broker.lastQuantity(t, "trading.buy_executed"); math.Abs(got-0.05) > 1e-12 {
			t.Errorf("Expected buy event quantity 0.05, got %.8f", got)
		}
	})
}

func TestResolveExecutedQuantity(t *testing.T) {
	testCases := []struct {
		name         string
		order        *binance.CreateOrderResponse
		formattedQty string
		adjustedQty  float64
		expected     float64
	}{
		{"uses executed quantity", &binance.CreateOrderResponse{ExecutedQuantity: "0.11000000"}, "0.12", 0.12, 0.11},
		{"falls back to sent quantity when executed is empty", &binance.CreateOrderResponse{}, "0.12", 0.12000000000000001, 0.12},
		{"falls back to sent quantity when executed is zero", &binance.CreateOrderResponse{ExecutedQuantity: "0.00000000"}, "0.12", 0.12, 0.12},
		{"falls back to sent quantity when order is nil", nil, "0.12", 0.12, 0.12},
		{"falls back to adjusted quantity when nothing parses", &binance.CreateOrderResponse{ExecutedQuantity: "abc"}, "", 0.12, 0.12},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveExecutedQuantity(tc.order, tc.formattedQty, tc.adjustedQty)
			if got != tc.expected {
				t.Errorf("Expected %.8f, got %.8f", tc.expected, got)
			}
		})
	}
}

func TestTradingBot_UseFixedQuantity(t *testing.T) {
	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := entity.NewMovingAverageStrategy(5, 20)

	t.Run("Fixed quantity mode", func(t *testing.T) {
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 100, "BRL", 0.1, 2.0, true)

		if !bot.GetUseFixedQuantity() {
			t.Error("Expected useFixedQuantity to be true")
		}
	})

	t.Run("Dynamic quantity mode", func(t *testing.T) {
		bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000, 100, "BRL", 0.1, 2.0, false)

		if bot.GetUseFixedQuantity() {
			t.Error("Expected useFixedQuantity to be false")
		}

		// Test setter
		bot.SetUseFixedQuantity(true)
		if !bot.GetUseFixedQuantity() {
			t.Error("Expected useFixedQuantity to be true after setting")
		}
	})
}

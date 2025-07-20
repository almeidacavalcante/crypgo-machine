package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	"testing"
	"time"
)

// MockTradeBotRepository implements repository.TradingBotRepository for testing
type MockTradeBotRepository struct{}

func (m *MockTradeBotRepository) Save(bot *entity.TradingBot) error           { return nil }
func (m *MockTradeBotRepository) Update(bot *entity.TradingBot) error         { return nil }
func (m *MockTradeBotRepository) Exists(id string) (bool, error)              { return false, nil }
func (m *MockTradeBotRepository) GetTradeByID(id string) (*entity.TradingBot, error) { return nil, nil }
func (m *MockTradeBotRepository) GetAllTradingBots() ([]*entity.TradingBot, error) { return nil, nil }
func (m *MockTradeBotRepository) GetTradingBotsByStatus(status entity.Status) ([]*entity.TradingBot, error) { return nil, nil }

// MockTradingDecisionLogRepository implements repository.TradingDecisionLogRepository for testing
type MockTradingDecisionLogRepository struct{}

func (m *MockTradingDecisionLogRepository) Save(log *entity.TradingDecisionLog) error { return nil }
func (m *MockTradingDecisionLogRepository) GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error) { return nil, nil }
func (m *MockTradingDecisionLogRepository) GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error) { return nil, nil }
func (m *MockTradingDecisionLogRepository) GetRecentLogs(limit int) ([]*entity.TradingDecisionLog, error) { return nil, nil }
func (m *MockTradingDecisionLogRepository) GetRecentLogsByDecision(decision string, limit int) ([]*entity.TradingDecisionLog, error) { return nil, nil }
func (m *MockTradingDecisionLogRepository) GetLogsWithFilters(decision string, symbol string, limit int, offset int) ([]*entity.TradingDecisionLog, int, error) { return nil, 0, nil }

// MockMessageBroker implements queue.MessageBroker for testing
type MockMessageBroker struct{}

func (m *MockMessageBroker) Publish(exchangeName string, message queue.Message) error { return nil }
func (m *MockMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error { return nil }
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
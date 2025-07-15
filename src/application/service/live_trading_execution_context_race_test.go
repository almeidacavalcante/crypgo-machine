package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	infraRepository "crypgo-machine/src/infra/repository"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockMessageBroker is a mock implementation of queue.MessageBroker
type MockMessageBroker struct{}

func (m *MockMessageBroker) Publish(exchangeName string, message queue.Message) error {
	return nil
}

func (m *MockMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	return nil
}

func (m *MockMessageBroker) Close() error {
	return nil
}

func TestLiveTradingExecutionContext_RaceConditionPrevention(t *testing.T) {
	// Setup
	binanceClient := external.NewBinanceClientFake()
	botRepo := infraRepository.NewTradeBotRepositoryInMemory()
	decisionRepo := infraRepository.NewTradingDecisionLogRepositoryInMemory()
	messageBroker := &MockMessageBroker{}
	
	context := NewLiveTradingExecutionContext(
		binanceClient,
		botRepo,
		decisionRepo,
		messageBroker,
		"binance",
	)

	// Create a test bot
	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := entity.NewMovingAverageStrategy(7, 25)
	bot := entity.NewTradingBot(symbol, 1000.0, strategy, 60, 10000.0, 1000.0, "BRL", 0.1, 2.0)
	
	// Save bot in repository
	botRepo.Save(bot)
	
	// Test concurrent BUY operations
	t.Run("Concurrent BUY operations should only execute once", func(t *testing.T) {
		var wg sync.WaitGroup
		var buySuccessCount int32
		var buyErrorCount int32
		
		// Reset bot state
		bot.GetOutOfPosition() // Ensure bot is not positioned
		botRepo.Update(bot)
		
		// Launch multiple concurrent BUY operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := context.ExecuteTrade(entity.Buy, bot, 650000.0, time.Now())
				if err != nil {
					t.Logf("BUY operation failed (expected): %v", err)
					atomic.AddInt32(&buyErrorCount, 1)
				} else {
					t.Logf("BUY operation succeeded")
					atomic.AddInt32(&buySuccessCount, 1)
				}
			}()
		}
		
		wg.Wait()
		
		// Verify only one BUY succeeded
		if atomic.LoadInt32(&buySuccessCount) != 1 {
			t.Errorf("Expected exactly 1 successful BUY, got %d", atomic.LoadInt32(&buySuccessCount))
		}
		
		// Verify bot is positioned after successful BUY
		freshBot, _ := botRepo.GetTradeByID(bot.Id.GetValue())
		if !freshBot.GetIsPositioned() {
			t.Error("Bot should be positioned after successful BUY")
		}
		
		t.Logf("Race condition test completed: %d successful, %d failed", atomic.LoadInt32(&buySuccessCount), atomic.LoadInt32(&buyErrorCount))
	})
	
	// Test concurrent SELL operations
	t.Run("Concurrent SELL operations should only execute once", func(t *testing.T) {
		var wg sync.WaitGroup
		var sellSuccessCount int32
		var sellErrorCount int32
		
		// Ensure bot is positioned for SELL test
		bot.GetIntoPosition()
		bot.SetEntryPrice(645000.0)
		botRepo.Update(bot)
		
		// Launch multiple concurrent SELL operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := context.ExecuteTrade(entity.Sell, bot, 655000.0, time.Now())
				if err != nil {
					t.Logf("SELL operation failed (expected): %v", err)
					atomic.AddInt32(&sellErrorCount, 1)
				} else {
					t.Logf("SELL operation succeeded")
					atomic.AddInt32(&sellSuccessCount, 1)
				}
			}()
		}
		
		wg.Wait()
		
		// Verify only one SELL succeeded
		if atomic.LoadInt32(&sellSuccessCount) != 1 {
			t.Errorf("Expected exactly 1 successful SELL, got %d", atomic.LoadInt32(&sellSuccessCount))
		}
		
		// Verify bot is not positioned after successful SELL
		freshBot, _ := botRepo.GetTradeByID(bot.Id.GetValue())
		if freshBot.GetIsPositioned() {
			t.Error("Bot should not be positioned after successful SELL")
		}
		
		t.Logf("Race condition test completed: %d successful, %d failed", atomic.LoadInt32(&sellSuccessCount), atomic.LoadInt32(&sellErrorCount))
	})
}

func TestLiveTradingExecutionContext_SequentialOperations(t *testing.T) {
	// Setup
	binanceClient := external.NewBinanceClientFake()
	botRepo := infraRepository.NewTradeBotRepositoryInMemory()
	decisionRepo := infraRepository.NewTradingDecisionLogRepositoryInMemory()
	messageBroker := &MockMessageBroker{}
	
	context := NewLiveTradingExecutionContext(
		binanceClient,
		botRepo,
		decisionRepo,
		messageBroker,
		"binance",
	)

	// Create a test bot
	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := entity.NewMovingAverageStrategy(7, 25)
	bot := entity.NewTradingBot(symbol, 1000.0, strategy, 60, 10000.0, 1000.0, "BRL", 0.1, 2.0)
	
	// Save bot in repository
	botRepo.Save(bot)
	
	t.Run("Sequential BUY-SELL operations should work correctly", func(t *testing.T) {
		// Reset bot state
		bot.GetOutOfPosition()
		botRepo.Update(bot)
		
		// Test BUY operation
		err := context.ExecuteTrade(entity.Buy, bot, 650000.0, time.Now())
		if err != nil {
			t.Fatalf("BUY operation failed: %v", err)
		}
		
		// Verify bot is positioned
		freshBot, _ := botRepo.GetTradeByID(bot.Id.GetValue())
		if !freshBot.GetIsPositioned() {
			t.Error("Bot should be positioned after BUY")
		}
		
		// Test duplicate BUY should fail
		err = context.ExecuteTrade(entity.Buy, bot, 651000.0, time.Now())
		if err == nil {
			t.Error("Duplicate BUY should have failed")
		}
		
		// Test SELL operation
		err = context.ExecuteTrade(entity.Sell, bot, 655000.0, time.Now())
		if err != nil {
			t.Fatalf("SELL operation failed: %v", err)
		}
		
		// Verify bot is not positioned
		freshBot, _ = botRepo.GetTradeByID(bot.Id.GetValue())
		if freshBot.GetIsPositioned() {
			t.Error("Bot should not be positioned after SELL")
		}
		
		// Test duplicate SELL should fail
		err = context.ExecuteTrade(entity.Sell, bot, 656000.0, time.Now())
		if err == nil {
			t.Error("Duplicate SELL should have failed")
		}
	})
}
package entity

import (
	"testing"

	vo "crypgo-machine/src/domain/vo"
)

func mustKline(close float64) vo.Kline {
	open := close - 0.5
	high := close + 0.5
	low := close - 1
	volume := 10.0
	closeTime := 1000
	k, err := vo.NewKline(open, close, high, low, volume, int64(closeTime))
	if err != nil {
		panic(err)
	}
	return k
}

func createTestBot() *TradingBot {
	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := NewMovingAverageStrategy(3, 5)
	bot := NewTradingBot(symbol, 0.001, strategy, 60)
	return bot
}

func TestMovingAverageStrategy_Buy(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(9), mustKline(8), mustKline(8), mustKline(8),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	bot := createTestBot()
	result := strategy.Decide(klines, bot)
	if result.Decision != Buy {
		t.Fatalf("expected Buy, got %s", result.Decision)
	}
}

func TestMovingAverageStrategy_Sell(t *testing.T) {
	klines := []vo.Kline{
		mustKline(8), mustKline(9), mustKline(10), mustKline(10), mustKline(10),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	bot := createTestBot()
	_ = bot.GetIntoPosition() // Bot needs to be positioned to sell
	
	// Set entry price lower than current price to ensure profit
	// Current price is 10, so set entry at 8 for positive profit
	bot.SetEntryPrice(8.0)
	
	result := strategy.Decide(klines, bot)
	if result.Decision != Sell {
		t.Fatalf("expected Sell, got %s", result.Decision)
	}
}

func TestMovingAverageStrategy_Hold(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(10), mustKline(10), mustKline(10), mustKline(10),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	bot := createTestBot()
	result := strategy.Decide(klines, bot)
	if result.Decision != Hold {
		t.Fatalf("expected Hold, got %s", result.Decision)
	}
}

func TestMovingAverageStrategy_NotEnoughData(t *testing.T) {
	klines := []vo.Kline{
		mustKline(10), mustKline(11),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	bot := createTestBot()
	result := strategy.Decide(klines, bot)
	if result.Decision != Hold {
		t.Fatalf("expected Hold due to insufficient data, got %s", result.Decision)
	}
}

func TestMovingAverageStrategy_NoSellAtLoss(t *testing.T) {
	// Test protection against selling at a loss
	klines := []vo.Kline{
		mustKline(12), mustKline(11), mustKline(10), mustKline(9), mustKline(8),
	}

	strategy := NewMovingAverageStrategy(3, 5)
	bot := createTestBot()
	_ = bot.GetIntoPosition() // Bot needs to be positioned
	
	// Set entry price higher than current price to simulate loss
	// Current price is 8, entry at 10 means 20% loss
	bot.SetEntryPrice(10.0)
	
	result := strategy.Decide(klines, bot)
	// Should HOLD instead of SELL to avoid loss
	if result.Decision != Hold {
		t.Fatalf("expected Hold (to avoid loss), got %s", result.Decision)
	}
	
	// Check that possible profit is tracked
	if profit, exists := result.AnalysisData["possibleProfit"]; exists {
		if profitFloat := profit.(float64); profitFloat >= 0 {
			t.Fatalf("expected negative profit, got %.2f", profitFloat)
		}
	}
}

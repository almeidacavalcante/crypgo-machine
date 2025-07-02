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

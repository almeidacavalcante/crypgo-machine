package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/domain/vo"
	"testing"
	"time"
)

func TestBacktestRSIStrategy_Oversold_Buy_Signal(t *testing.T) {
	// Create RSI strategy
	rsiParams := service.RSIParams{
		Period:              14,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	strategy, err := service.NewTradeStrategyFactory("RSI", rsiParams)
	if err != nil {
		t.Fatalf("Failed to create RSI strategy: %v", err)
	}

	// Create test bot
	symbol, _ := vo.NewSymbol("BTCUSDT")
	bot := entity.NewTradingBot(
		symbol,
		0.001,       // quantity
		strategy,
		300,         // intervalSeconds
		1000.0,      // initialCapital
		100.0,       // tradeAmount
		"USDT",      // currency
		0.1,         // tradingFees
		1.0,         // minimumProfitThreshold
		true,        // useFixedQuantity
	)

	// Create klines that should trigger oversold condition (RSI < 30)
	klines := createRSITestKlines([]float64{
		// Start high and drop significantly to create oversold condition
		50.0, 49.0, 48.0, 47.0, 46.0, 45.0, 44.0, 43.0, 42.0, 41.0,
		40.0, 39.0, 38.0, 37.0, 36.0, 35.0, 34.0, 33.0, 32.0, 31.0,
	})

	// Test RSI calculation and decision
	result := strategy.Decide(klines, bot)

	t.Logf("RSI Strategy Decision: %s", result.Decision)
	t.Logf("RSI Value: %v", result.AnalysisData["rsi"])
	t.Logf("RSI Signal: %v", result.AnalysisData["signal"])
	t.Logf("Reason: %v", result.AnalysisData["reason"])

	// Verify that the strategy recognizes oversold condition
	if rsiValue, ok := result.AnalysisData["rsi"].(float64); ok {
		if rsiValue >= 30.0 {
			t.Errorf("Expected RSI to be oversold (< 30), got: %.2f", rsiValue)
		}
	}

	// Should generate buy signal when not positioned and oversold
	if !bot.GetIsPositioned() && result.Decision != entity.Buy {
		t.Errorf("Expected Buy decision for oversold RSI when not positioned, got: %s", result.Decision)
	}
}

func TestBacktestRSIStrategy_Overbought_Signal_Recognition(t *testing.T) {
	// Create RSI strategy
	rsiParams := service.RSIParams{
		Period:              14,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	strategy, err := service.NewTradeStrategyFactory("RSI", rsiParams)
	if err != nil {
		t.Fatalf("Failed to create RSI strategy: %v", err)
	}

	// Create test bot (not positioned)
	symbol, _ := vo.NewSymbol("BTCUSDT")
	bot := entity.NewTradingBot(
		symbol,
		0.001,       // quantity
		strategy,
		300,         // intervalSeconds
		1000.0,      // initialCapital
		100.0,       // tradeAmount
		"USDT",      // currency
		0.1,         // tradingFees
		1.0,         // minimumProfitThreshold
		true,        // useFixedQuantity
	)

	// Create klines that should trigger overbought condition (RSI > 70)
	klines := createRSITestKlines([]float64{
		// Start low and rise significantly to create overbought condition
		50.0, 51.0, 52.0, 53.0, 54.0, 55.0, 56.0, 57.0, 58.0, 59.0,
		60.0, 61.0, 62.0, 63.0, 64.0, 65.0, 66.0, 67.0, 68.0, 80.0, // Big jump at end
	})

	// Test RSI calculation and decision
	result := strategy.Decide(klines, bot)

	t.Logf("RSI Strategy Decision: %s", result.Decision)
	t.Logf("RSI Value: %v", result.AnalysisData["rsi"])
	t.Logf("RSI Signal: %v", result.AnalysisData["signal"])
	t.Logf("Reason: %v", result.AnalysisData["reason"])

	// Verify that the strategy recognizes overbought condition
	if rsiValue, ok := result.AnalysisData["rsi"].(float64); ok {
		if rsiValue <= 70.0 {
			t.Errorf("Expected RSI to be overbought (> 70), got: %.2f", rsiValue)
		}
	}

	// When not positioned and overbought, should wait
	if !bot.GetIsPositioned() && result.Decision == entity.Hold {
		t.Logf("✅ RSI strategy correctly identified overbought condition and held position")
	}
}

func TestBacktestRSIStrategy_BasicCalculation(t *testing.T) {
	// Create RSI strategy
	rsiParams := service.RSIParams{
		Period:              14,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	strategy, err := service.NewTradeStrategyFactory("RSI", rsiParams)
	if err != nil {
		t.Fatalf("Failed to create RSI strategy: %v", err)
	}

	// Create test bot
	symbol, _ := vo.NewSymbol("BTCUSDT")
	bot := entity.NewTradingBot(
		symbol,
		0.001,       // quantity
		strategy,
		300,         // intervalSeconds
		1000.0,      // initialCapital
		100.0,       // tradeAmount
		"USDT",      // currency
		0.1,         // tradingFees
		1.0,         // minimumProfitThreshold
		true,        // useFixedQuantity
	)

	// Create neutral klines for basic RSI calculation test
	klines := createRSITestKlines([]float64{
		50.0, 50.5, 51.0, 50.8, 51.2, 50.9, 51.1, 50.7, 51.3, 50.6,
		51.4, 50.5, 51.5, 50.4, 51.6, 50.3, 51.7, 50.2, 51.8, 50.1,
	})

	// Test RSI decision with test data
	result := strategy.Decide(klines, bot)

	t.Logf("RSI Strategy with test data:")
	t.Logf("  Decision: %s", result.Decision)
	t.Logf("  RSI Value: %v", result.AnalysisData["rsi"])
	t.Logf("  Signal: %v", result.AnalysisData["signal"])
	t.Logf("  Reason: %v", result.AnalysisData["reason"])

	// Verify RSI calculation is working
	if rsiValue, ok := result.AnalysisData["rsi"].(float64); ok {
		if rsiValue < 0 || rsiValue > 100 {
			t.Errorf("RSI value should be between 0 and 100, got: %.2f", rsiValue)
		}
	} else {
		t.Error("RSI value not found in analysis data")
	}

	// Verify signal classification
	if signal, ok := result.AnalysisData["signal"].(string); ok {
		validSignals := []string{"OVERSOLD", "OVERBOUGHT", "NEUTRAL"}
		found := false
		for _, validSignal := range validSignals {
			if signal == validSignal {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Invalid RSI signal: %s", signal)
		}
	}

	t.Log("✅ RSI strategy basic calculation test completed successfully")
}

// Helper function to create test klines for RSI testing
func createRSITestKlines(closePrices []float64) []vo.Kline {
	klines := make([]vo.Kline, len(closePrices))
	baseTime := time.Now().Unix() * 1000

	for i, price := range closePrices {
		kline, _ := vo.NewKline(
			price,      // open
			price,      // close
			price+0.5,  // high
			price-0.5,  // low
			1000.0,     // volume
			baseTime+int64(i*300000), // closeTime (5 minute intervals)
		)
		klines[i] = kline
	}

	return klines
}
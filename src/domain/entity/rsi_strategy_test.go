package entity

import (
	"crypgo-machine/src/domain/vo"
	"testing"
	"time"
)

func TestRSIStrategy_Decide_Success(t *testing.T) {
	strategy := NewRSIStrategy(14)

	// Create test trading bot
	bot := createTestTradingBot(false, 0.0, 1.0) // not positioned, no entry price, 1% min profit

	// Test data that should result in oversold condition (RSI < 30)
	klines := createTestKlinesForRSI([]float64{
		55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40,
	})

	result := strategy.Decide(klines, bot)

	if result.Decision != Buy {
		t.Errorf("Expected Buy decision for oversold RSI, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] != "rsi_oversold_buy_signal" {
		t.Errorf("Expected oversold reason, got: %s", result.AnalysisData["reason"])
	}
}

func TestRSIStrategy_Decide_InsufficientData(t *testing.T) {
	strategy := NewRSIStrategy(14)
	bot := createTestTradingBot(false, 0.0, 1.0)

	// Only 10 klines for period 14
	klines := createTestKlinesForRSI([]float64{44, 44.34, 44.09, 44.15, 43.61, 44.33, 44.83, 45.85, 46.08, 45.89})

	result := strategy.Decide(klines, bot)

	if result.Decision != Hold {
		t.Errorf("Expected Hold decision for insufficient data, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] != "insufficient_data" {
		t.Errorf("Expected insufficient_data reason, got: %s", result.AnalysisData["reason"])
	}
}

func TestRSIStrategy_Decide_OverboughtSell(t *testing.T) {
	strategy := NewRSIStrategy(14)

	// Create positioned bot with sufficient profit
	bot := createTestTradingBot(true, 40.0, 1.0) // positioned, entry at 40, 1% min profit

	// Test data that should result in overbought condition (RSI > 70)
	klines := createTestKlinesForRSI([]float64{
		40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	})

	result := strategy.Decide(klines, bot)

	if result.Decision != Sell {
		t.Errorf("Expected Sell decision for overbought RSI with profit, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] != "rsi_overbought_sell_with_profit" {
		t.Errorf("Expected overbought sell reason, got: %s", result.AnalysisData["reason"])
	}
}

func TestRSIStrategy_Decide_OverboughtHoldInsufficientProfit(t *testing.T) {
	strategy := NewRSIStrategy(14)

	// Create positioned bot with insufficient profit (entry at 54, current at 55, less than 1% profit)
	bot := createTestTradingBot(true, 54.0, 5.0) // positioned, entry at 54, 5% min profit

	// Test data that should result in overbought condition (RSI > 70)
	klines := createTestKlinesForRSI([]float64{
		40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	})

	result := strategy.Decide(klines, bot)

	if result.Decision != Hold {
		t.Errorf("Expected Hold decision for overbought RSI with insufficient profit, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] != "rsi_overbought_hold_insufficient_profit" {
		t.Errorf("Expected overbought hold reason, got: %s", result.AnalysisData["reason"])
	}
}

func TestRSIStrategy_GetName(t *testing.T) {
	strategy := NewRSIStrategy(14)

	if strategy.GetName() != "RSI" {
		t.Errorf("Expected strategy name 'RSI', got: %s", strategy.GetName())
	}
}

func TestRSIStrategy_GetParams(t *testing.T) {
	strategy := NewRSIStrategy(14)
	params := strategy.GetParams()

	if params["Period"] != 14 {
		t.Errorf("Expected Period 14, got: %v", params["Period"])
	}

	if params["OversoldThreshold"] != 30.0 {
		t.Errorf("Expected OversoldThreshold 30.0, got: %v", params["OversoldThreshold"])
	}

	if params["OverboughtThreshold"] != 70.0 {
		t.Errorf("Expected OverboughtThreshold 70.0, got: %v", params["OverboughtThreshold"])
	}
}

func TestRSIStrategy_CustomThresholds(t *testing.T) {
	minimumSpread, _ := vo.NewMinimumSpread(0.5)
	strategy := NewRSIStrategyWithCustomThresholds(14, 25.0, 75.0, minimumSpread)

	params := strategy.GetParams()

	if params["OversoldThreshold"] != 25.0 {
		t.Errorf("Expected custom OversoldThreshold 25.0, got: %v", params["OversoldThreshold"])
	}

	if params["OverboughtThreshold"] != 75.0 {
		t.Errorf("Expected custom OverboughtThreshold 75.0, got: %v", params["OverboughtThreshold"])
	}

	if params["MinimumSpread"] != 0.5 {
		t.Errorf("Expected custom MinimumSpread 0.5, got: %v", params["MinimumSpread"])
	}
}

func TestRSIStrategy_Stoploss_Trigger(t *testing.T) {
	minimumSpread, _ := vo.NewMinimumSpread(0.1)
	strategy := NewRSIStrategyWithStoploss(14, 30.0, 70.0, minimumSpread, 5.0) // 5% stoploss

	// Create positioned bot with entry at 50, current price will be 47 (6% loss)
	bot := createTestTradingBot(true, 50.0, 1.0) // positioned, entry at 50, 1% min profit

	// Create klines with declining prices (should trigger stoploss)
	klines := createTestKlinesForRSI([]float64{
		50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 47, // Final price 47 = 6% loss
	})

	result := strategy.Decide(klines, bot)

	if result.Decision != Sell {
		t.Errorf("Expected Sell decision for stoploss trigger, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] != "stoploss_triggered" {
		t.Errorf("Expected stoploss_triggered reason, got: %s", result.AnalysisData["reason"])
	}

	if result.AnalysisData["stoplossThreshold"] != 5.0 {
		t.Errorf("Expected stoploss threshold 5.0, got: %v", result.AnalysisData["stoplossThreshold"])
	}
}

func TestRSIStrategy_Stoploss_NoTrigger(t *testing.T) {
	minimumSpread, _ := vo.NewMinimumSpread(0.1)
	strategy := NewRSIStrategyWithStoploss(14, 30.0, 70.0, minimumSpread, 5.0) // 5% stoploss

	// Create positioned bot with entry at 50, current price will be 48 (4% loss, below stoploss)
	bot := createTestTradingBot(true, 50.0, 1.0) // positioned, entry at 50, 1% min profit

	// Create klines with mild decline (should not trigger stoploss)
	klines := createTestKlinesForRSI([]float64{
		50, 49, 48, 47, 46, 47, 48, 49, 50, 49, 48, 47, 48, 49, 50, 48, // Final price 48 = 4% loss
	})

	result := strategy.Decide(klines, bot)

	if result.Decision == Sell {
		t.Errorf("Expected no Sell decision for insufficient stoploss, got: %s", result.Decision)
	}

	if result.AnalysisData["reason"] == "stoploss_triggered" {
		t.Errorf("Should not trigger stoploss for 4%% loss with 5%% threshold")
	}
}

// Helper functions
func createTestTradingBot(isPositioned bool, entryPrice, minProfitThreshold float64) *TradingBot {
	symbol, _ := vo.NewSymbol("BTCUSDT")
	quantity := 0.001
	strategy := NewRSIStrategy(14)
	
	bot := NewTradingBot(
		symbol,
		quantity,
		strategy,
		300,      // intervalSeconds
		1000.0,   // initialCapital
		100.0,    // tradeAmount
		"USDT",   // currency
		0.1,      // tradingFees
		minProfitThreshold,
		true,     // useFixedQuantity
	)
	
	// Set bot state for testing
	if isPositioned {
		bot.isPositioned = true
		bot.entryPrice = entryPrice
	}
	
	return bot
}

func createTestKlinesForRSI(closePrices []float64) []vo.Kline {
	klines := make([]vo.Kline, len(closePrices))
	baseTime := time.Now().Unix() * 1000

	for i, price := range closePrices {
		kline, _ := vo.NewKline(
			price,      // open
			price,      // close
			price+0.1,  // high
			price-0.1,  // low
			1000.0,     // volume
			baseTime+int64(i*60000), // closeTime (1 minute intervals)
		)
		klines[i] = kline
	}

	return klines
}
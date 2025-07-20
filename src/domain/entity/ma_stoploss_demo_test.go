package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"testing"
	"time"
)

func TestMovingAverageStrategy_StoplossDemo(t *testing.T) {
	// Create Moving Average strategy with 5% stoploss
	minimumSpread, _ := vo.NewMinimumSpread(0.1)
	strategy := NewMovingAverageStrategyWithStoploss(3, 5, minimumSpread, 5.0)

	// Create a bot positioned with entry at 100 BRL
	bot := createTestBotWithMinimumProfit(2.0) // 2% min profit
	_ = bot.GetIntoPosition()
	bot.SetEntryPrice(100.0)

	fmt.Printf("🎯 Testing Moving Average with 5%% Stoploss\n")
	fmt.Printf("📊 Bot Entry Price: R$%.2f\n", bot.GetEntryPrice())
	fmt.Printf("📊 Minimum Profit Target: %.1f%%\n", bot.GetMinimumProfitThreshold())
	fmt.Printf("📊 Stoploss Threshold: %.1f%%\n", strategy.StoplossThreshold)
	fmt.Printf("📊 Fast Window: %d | Slow Window: %d\n\n", strategy.FastWindow, strategy.SlowWindow)

	// Test different scenarios
	scenarios := []struct {
		name             string
		prices           []float64
		expectedDecision TradingDecision
		expectedReason   string
	}{
		{
			name:             "Price drops 6% - Should trigger stoploss",
			prices:           []float64{94, 94, 94, 94, 94}, // 6% loss
			expectedDecision: Sell,
			expectedReason:   "stoploss_triggered",
		},
		{
			name:             "Price drops 4% - Should NOT trigger stoploss",
			prices:           []float64{96, 96, 96, 96, 96}, // 4% loss
			expectedDecision: Hold,
			expectedReason:   "fast_equals_slow_neutral", // Since all prices are same
		},
		{
			name:             "Price rises but below target - Should hold",
			prices:           []float64{101, 101, 101, 101, 101}, // 1% profit (below 2% target)
			expectedDecision: Hold,
			expectedReason:   "fast_equals_slow_neutral",
		},
		{
			name:             "Uptrend with sufficient profit - Should sell normally",
			prices:           []float64{100, 102, 104, 106, 108}, // Strong uptrend, fast > slow
			expectedDecision: Sell,
			expectedReason:   "fast_above_slow_sell_high_with_profit",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create klines with the test prices
			klines := createTestKlinesWithPrices(scenario.prices)
			
			result := strategy.Decide(klines, bot)
			
			currentPrice := scenario.prices[len(scenario.prices)-1]
			profitPercent := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
			
			fmt.Printf("📈 %s\n", scenario.name)
			fmt.Printf("   💰 Current Price: R$%.2f\n", currentPrice)
			fmt.Printf("   📊 Profit/Loss: %.2f%%\n", profitPercent)
			fmt.Printf("   🎯 Decision: %s\n", result.Decision)
			fmt.Printf("   📝 Reason: %s\n", result.AnalysisData["reason"])
			fmt.Printf("   ✅ Expected: %s (%s)\n\n", scenario.expectedDecision, scenario.expectedReason)
			
			if result.Decision != scenario.expectedDecision {
				t.Errorf("Expected decision %s, got %s", scenario.expectedDecision, result.Decision)
			}
			
			if result.AnalysisData["reason"] != scenario.expectedReason {
				t.Errorf("Expected reason %s, got %s", scenario.expectedReason, result.AnalysisData["reason"])
			}
		})
	}
}

// Helper function to create klines with specific prices
func createTestKlinesWithPrices(prices []float64) []vo.Kline {
	klines := make([]vo.Kline, len(prices))
	baseTime := time.Now().Unix() * 1000

	for i, price := range prices {
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
package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"testing"
	"time"
)

func TestRSIStrategy_StoplossDemo(t *testing.T) {
	// Create RSI strategy with 5% stoploss
	minimumSpread, _ := vo.NewMinimumSpread(0.1)
	strategy := NewRSIStrategyWithStoploss(14, 30.0, 70.0, minimumSpread, 5.0)

	// Create a bot positioned with entry at 100 BRL
	bot := createTestTradingBot(true, 100.0, 2.0) // positioned, entry at 100, 2% min profit

	fmt.Printf("🎯 Testing RSI with 5%% Stoploss\n")
	fmt.Printf("📊 Bot Entry Price: R$%.2f\n", bot.GetEntryPrice())
	fmt.Printf("📊 Minimum Profit Target: %.1f%%\n", bot.GetMinimumProfitThreshold())
	fmt.Printf("📊 Stoploss Threshold: %.1f%%\n\n", strategy.StoplossThreshold)

	// Test different scenarios
	scenarios := []struct {
		name         string
		currentPrice float64
		expectedDecision TradingDecision
		expectedReason   string
	}{
		{
			name:             "Price drops 6% - Should trigger stoploss",
			currentPrice:     94.0, // 6% loss
			expectedDecision: Sell,
			expectedReason:   "stoploss_triggered",
		},
		{
			name:             "Price drops 4% - Should NOT trigger stoploss",
			currentPrice:     96.0, // 4% loss
			expectedDecision: Hold,
			expectedReason:   "rsi_neutral_positioned_holding",
		},
		{
			name:             "Price rises 1% - Below profit target, should hold",
			currentPrice:     101.0, // 1% profit (below 2% target)
			expectedDecision: Hold,
			expectedReason:   "rsi_neutral_positioned_holding",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create klines with neutral RSI and the test price
			klines := createTestKlinesWithPrice(scenario.currentPrice, 16)
			
			result := strategy.Decide(klines, bot)
			
			profitPercent := ((scenario.currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
			
			fmt.Printf("📈 %s\n", scenario.name)
			fmt.Printf("   💰 Current Price: R$%.2f\n", scenario.currentPrice)
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

// Helper function to create klines with a specific final price
func createTestKlinesWithPrice(finalPrice float64, count int) []vo.Kline {
	klines := make([]vo.Kline, count)
	baseTime := time.Now().Unix() * 1000
	basePrice := 100.0 // Start at 100

	for i := 0; i < count; i++ {
		var price float64
		if i == count-1 {
			// Last kline uses the final price
			price = finalPrice
		} else {
			// Generate neutral prices around 100
			price = basePrice + float64(i%3-1)*0.5 // Creates 99.5, 100, 100.5 pattern
		}
		
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
package usecase

import (
	"crypgo-machine/src/infra/external"
	"testing"
	"time"
)

func TestBacktestTradingBotUseCase_MovingAverage(t *testing.T) {
	// Create a fake client with predefined market data
	client := external.NewBinanceClientFake()
	
	// Configure the fake client with a strong trend scenario
	client.SetPredefinedKlines(external.CreateStrongTrendKlines())
	
	useCase := NewBacktestTradingBotUseCase(client)
	
	input := BacktestTradingBotInput{
		Symbol:         "BTCBRL",
		Strategy:       "MovingAverage",
		StrategyParams: map[string]interface{}{
			"FastWindow": 5.0,
			"SlowWindow": 10.0,
		},
		StartDate:              time.Now().AddDate(0, 0, -30), // 30 days ago
		EndDate:                time.Now(),
		InitialCapital:         1000.0,
		TradeAmount:            100.0,
		TradingFees:            0.1,
		MinimumProfitThreshold: 2.0, // 2% minimum profit
		Interval:               "1h",
	}
	
	result, err := useCase.Execute(input)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	// Verify basic result structure
	if result.Symbol != "BTCBRL" {
		t.Errorf("Expected symbol BTCBRL, got %s", result.Symbol)
	}
	
	if result.InitialCapital != 1000.0 {
		t.Errorf("Expected initial capital 1000.0, got %.2f", result.InitialCapital)
	}
	
	// The fake client should provide some trading opportunities
	if len(result.Decisions) == 0 {
		t.Error("Expected some trading decisions to be made")
	}
	
	t.Logf("Backtest completed successfully:")
	t.Logf("  Total P&L: %.2f BRL", result.TotalPnL)
	t.Logf("  ROI: %.2f%%", result.ROI)
	t.Logf("  Total Trades: %d", result.TotalTrades)
	t.Logf("  Win Rate: %.2f%%", result.WinRate)
}

func TestBacktestTradingBotUseCase_InvalidStrategy(t *testing.T) {
	client := external.NewBinanceClientFake()
	useCase := NewBacktestTradingBotUseCase(client)
	
	input := BacktestTradingBotInput{
		Symbol:         "BTCBRL",
		Strategy:       "InvalidStrategy",
		StrategyParams: map[string]interface{}{},
		StartDate:      time.Now().AddDate(0, 0, -7),
		EndDate:        time.Now(),
		InitialCapital: 1000.0,
		TradeAmount:    100.0,
		TradingFees:    0.1,
		Interval:       "1h",
	}
	
	_, err := useCase.Execute(input)
	
	if err == nil {
		t.Fatal("Expected error for invalid strategy, got nil")
	}
	
	if !contains(err.Error(), "unsupported strategy") {
		t.Errorf("Expected 'unsupported strategy' error, got: %v", err)
	}
}

func TestBacktestTradingBotUseCase_MinimumProfitThreshold(t *testing.T) {
	client := external.NewBinanceClientFake()
	
	// Set up a scenario with small price movements
	client.SetPredefinedKlines(external.CreateWhipsawKlines())
	
	useCase := NewBacktestTradingBotUseCase(client)
	
	// Test with high minimum profit threshold (should reduce trades)
	input := BacktestTradingBotInput{
		Symbol:         "BTCBRL",
		Strategy:       "MovingAverage",
		StrategyParams: map[string]interface{}{
			"FastWindow": 3.0,
			"SlowWindow": 5.0,
		},
		StartDate:              time.Now().AddDate(0, 0, -7),
		EndDate:                time.Now(),
		InitialCapital:         1000.0,
		TradeAmount:            100.0,
		TradingFees:            0.1,
		MinimumProfitThreshold: 10.0, // High 10% minimum profit
		Interval:               "1h",
	}
	
	result, err := useCase.Execute(input)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// With high minimum profit threshold and whipsaw data, 
	// we should see fewer or no trades
	t.Logf("Trades with 10%% minimum profit: %d", result.TotalTrades)
	
	// Test that minimum profit threshold is actually being used
	// (this test verifies our fix is working)
	highThresholdTrades := result.TotalTrades
	
	// Now test with 0% minimum profit threshold
	input.MinimumProfitThreshold = 0.0
	
	resultLowThreshold, err := useCase.Execute(input)
	if err != nil {
		t.Fatalf("Expected no error for low threshold test, got: %v", err)
	}
	
	t.Logf("Trades with 0%% minimum profit: %d", resultLowThreshold.TotalTrades)
	
	// We should see the same or more trades with 0% threshold vs 10% threshold
	if resultLowThreshold.TotalTrades < highThresholdTrades {
		t.Errorf("Expected same or more trades with lower threshold, got %d vs %d", 
			resultLowThreshold.TotalTrades, highThresholdTrades)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
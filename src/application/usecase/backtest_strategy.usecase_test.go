package usecase

import (
	"crypgo-machine/src/domain/vo"
	"testing"
	"time"
)

func TestBacktestStrategyUseCase_Execute_Success(t *testing.T) {
	useCase := NewBacktestStrategyUseCase()

	// Create test historical data
	historicalData := createTestHistoricalData()

	input := InputBacktestStrategy{
		StrategyName:   "MovingAverage",
		Symbol:         "BTCBRL",
		HistoricalData: historicalData,
		InitialCapital: 10000.0,
		Currency:       "BRL",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		TradingFees:    0.1,
	}

	result, err := useCase.Execute(input)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Validate basic result properties
	if result.GetStrategyName() != "MovingAverage" {
		t.Errorf("Expected strategy name 'MovingAverage', got: %s", result.GetStrategyName())
	}

	if result.GetSymbol().GetValue() != "BTCBRL" {
		t.Errorf("Expected symbol 'BTCBRL', got: %s", result.GetSymbol().GetValue())
	}

	if result.GetInitialCapital() != 10000.0 {
		t.Errorf("Expected initial capital 10000.0, got: %f", result.GetInitialCapital())
	}

	// Capital history should have at least initial value
	if len(result.GetCapitalHistory()) == 0 {
		t.Error("Expected capital history to have at least one entry")
	}

	// Check that final capital is calculated
	if result.GetFinalCapital() == 0 {
		t.Error("Expected final capital to be calculated")
	}
}

func TestBacktestStrategyUseCase_Execute_InvalidStrategy(t *testing.T) {
	useCase := NewBacktestStrategyUseCase()

	input := InputBacktestStrategy{
		StrategyName:   "InvalidStrategy",
		Symbol:         "BTCBRL",
		HistoricalData: createTestHistoricalData(),
		InitialCapital: 10000.0,
		Currency:       "BRL",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		TradingFees:    0.1,
	}

	result, err := useCase.Execute(input)

	if err == nil {
		t.Fatal("Expected error for invalid strategy, got none")
	}

	if result != nil {
		t.Fatal("Expected nil result for invalid strategy")
	}
}

func TestBacktestStrategyUseCase_Execute_InvalidInput(t *testing.T) {
	useCase := NewBacktestStrategyUseCase()

	tests := []struct {
		name  string
		input InputBacktestStrategy
	}{
		{
			name: "Empty strategy name",
			input: InputBacktestStrategy{
				StrategyName:   "",
				Symbol:         "BTCBRL",
				HistoricalData: createTestHistoricalData(),
				InitialCapital: 10000.0,
				Currency:       "BRL",
				StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				TradingFees:    0.1,
			},
		},
		{
			name: "Empty symbol",
			input: InputBacktestStrategy{
				StrategyName:   "MovingAverage",
				Symbol:         "",
				HistoricalData: createTestHistoricalData(),
				InitialCapital: 10000.0,
				Currency:       "BRL",
				StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				TradingFees:    0.1,
			},
		},
		{
			name: "Negative initial capital",
			input: InputBacktestStrategy{
				StrategyName:   "MovingAverage",
				Symbol:         "BTCBRL",
				HistoricalData: createTestHistoricalData(),
				InitialCapital: -1000.0,
				Currency:       "BRL",
				StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				TradingFees:    0.1,
			},
		},
		{
			name: "Empty historical data",
			input: InputBacktestStrategy{
				StrategyName:   "MovingAverage",
				Symbol:         "BTCBRL",
				HistoricalData: []vo.Kline{},
				InitialCapital: 10000.0,
				Currency:       "BRL",
				StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				TradingFees:    0.1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := useCase.Execute(tt.input)

			if err == nil {
				t.Fatalf("Expected error for %s, got none", tt.name)
			}

			if result != nil {
				t.Fatalf("Expected nil result for %s", tt.name)
			}
		})
	}
}

func TestBacktestStrategyUseCase_BreakoutStrategy(t *testing.T) {
	useCase := NewBacktestStrategyUseCase()

	// Create test historical data with breakout pattern
	historicalData := createBreakoutTestData()

	input := InputBacktestStrategy{
		StrategyName:   "Breakout",
		Symbol:         "BTCBRL",
		HistoricalData: historicalData,
		InitialCapital: 5000.0,
		Currency:       "BRL",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		TradingFees:    0.05,
	}

	result, err := useCase.Execute(input)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.GetStrategyName() != "Breakout" {
		t.Errorf("Expected strategy name 'Breakout', got: %s", result.GetStrategyName())
	}
}

// Helper functions for test data
func createTestHistoricalData() []vo.Kline {
	var klines []vo.Kline
	basePrice := 100.0
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create 50 klines with trending data
	for i := 0; i < 50; i++ {
		price := basePrice + float64(i)*0.5 // Gradual uptrend
		kline, _ := vo.NewKline(
			price-0.2,    // open
			price+0.3,    // close
			price+0.5,    // high
			price-0.3,    // low
			1000.0,       // volume
			baseTime.Add(time.Hour*time.Duration(i)).UnixMilli(),
		)
		klines = append(klines, kline)
	}

	return klines
}

func createBreakoutTestData() []vo.Kline {
	var klines []vo.Kline
	basePrice := 200.0
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create sideways market first (30 periods)
	for i := 0; i < 30; i++ {
		price := basePrice + (float64(i%5)-2)*0.1 // Sideways movement
		kline, _ := vo.NewKline(
			price-0.1,
			price+0.1,
			price+0.2,
			price-0.2,
			1000.0,
			baseTime.Add(time.Hour*time.Duration(i)).UnixMilli(),
		)
		klines = append(klines, kline)
	}

	// Create breakout (20 periods)
	for i := 30; i < 50; i++ {
		price := basePrice + float64(i-30)*2.0 // Strong breakout
		kline, _ := vo.NewKline(
			price-0.5,
			price+1.0,
			price+1.5,
			price-0.5,
			2000.0,
			baseTime.Add(time.Hour*time.Duration(i)).UnixMilli(),
		)
		klines = append(klines, kline)
	}

	return klines
}
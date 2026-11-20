package service

import (
	"testing"
)

func TestNewTradeStrategyFactory_RSI_Success(t *testing.T) {
	params := RSIParams{
		Period:              14,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	strategy, err := NewTradeStrategyFactory("RSI", params)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if strategy == nil {
		t.Fatal("Expected strategy, got nil")
	}

	if strategy.GetName() != "RSI" {
		t.Errorf("Expected strategy name 'RSI', got: %s", strategy.GetName())
	}

	strategyParams := strategy.GetParams()
	if strategyParams["Period"] != 14 {
		t.Errorf("Expected Period 14, got: %v", strategyParams["Period"])
	}
}

func TestNewTradeStrategyFactory_RSI_InvalidPeriod(t *testing.T) {
	params := RSIParams{
		Period:              0, // Invalid period
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	_, err := NewTradeStrategyFactory("RSI", params)

	if err == nil {
		t.Fatal("Expected error for invalid period, got nil")
	}

	expectedMessage := "missing or invalid fields for RSI: Period must be > 0"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_RSI_InvalidThresholds(t *testing.T) {
	params := RSIParams{
		Period:              14,
		OversoldThreshold:   -5.0, // Invalid threshold (zero means "use default")
		OverboughtThreshold: 70.0,
	}

	_, err := NewTradeStrategyFactory("RSI", params)

	if err == nil {
		t.Fatal("Expected error for invalid threshold, got nil")
	}

	expectedMessage := "OversoldThreshold must be between 0 and 100"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_RSI_ThresholdLogicError(t *testing.T) {
	params := RSIParams{
		Period:              14,
		OversoldThreshold:   80.0, // Greater than overbought
		OverboughtThreshold: 70.0,
	}

	_, err := NewTradeStrategyFactory("RSI", params)

	if err == nil {
		t.Fatal("Expected error for invalid threshold logic, got nil")
	}

	expectedMessage := "OversoldThreshold must be less than OverboughtThreshold"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_RSI_CustomThresholds(t *testing.T) {
	params := RSIParams{
		Period:              14,
		OversoldThreshold:   25.0, // Custom threshold
		OverboughtThreshold: 75.0, // Custom threshold
	}

	strategy, err := NewTradeStrategyFactory("RSI", params)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	strategyParams := strategy.GetParams()
	if strategyParams["OversoldThreshold"] != 25.0 {
		t.Errorf("Expected OversoldThreshold 25.0, got: %v", strategyParams["OversoldThreshold"])
	}

	if strategyParams["OverboughtThreshold"] != 75.0 {
		t.Errorf("Expected OverboughtThreshold 75.0, got: %v", strategyParams["OverboughtThreshold"])
	}
}

func TestNewTradeStrategyFactory_RSI_WrongParamsType(t *testing.T) {
	params := MovingAverageParams{ // Wrong params type
		FastWindow: 5,
		SlowWindow: 10,
	}

	_, err := NewTradeStrategyFactory("RSI", params)

	if err == nil {
		t.Fatal("Expected error for wrong params type, got nil")
	}

	expectedMessage := "params must be RSIParams for RSI strategy"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_MovingAverage_StillWorks(t *testing.T) {
	params := MovingAverageParams{
		FastWindow: 5,
		SlowWindow: 10,
	}

	strategy, err := NewTradeStrategyFactory("MovingAverage", params)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if strategy.GetName() != "MovingAverage" {
		t.Errorf("Expected strategy name 'MovingAverage', got: %s", strategy.GetName())
	}
}

func TestNewTradeStrategyFactory_UnknownStrategy(t *testing.T) {
	params := RSIParams{
		Period:              14,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
	}

	_, err := NewTradeStrategyFactory("UnknownStrategy", params)

	if err == nil {
		t.Fatal("Expected error for unknown strategy, got nil")
	}

	expectedMessage := "unknown or invalid strategy: UnknownStrategy"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_RSI_OmittedThresholdsUseDefaults(t *testing.T) {
	params := RSIParams{Period: 14} // thresholds omitted

	strategy, err := NewTradeStrategyFactory("RSI", params)
	if err != nil {
		t.Fatalf("Expected no error when thresholds are omitted, got: %v", err)
	}

	strategyParams := strategy.GetParams()
	if strategyParams["OversoldThreshold"] != 30.0 {
		t.Errorf("Expected default OversoldThreshold 30.0, got: %v", strategyParams["OversoldThreshold"])
	}
	if strategyParams["OverboughtThreshold"] != 70.0 {
		t.Errorf("Expected default OverboughtThreshold 70.0, got: %v", strategyParams["OverboughtThreshold"])
	}
}

func TestNewTradeStrategyFactory_RSI_OnlyOneThresholdOmitted(t *testing.T) {
	strategy, err := NewTradeStrategyFactory("RSI", RSIParams{Period: 14, OversoldThreshold: 25.0})
	if err != nil {
		t.Fatalf("Expected no error when OverboughtThreshold is omitted, got: %v", err)
	}
	p := strategy.GetParams()
	if p["OversoldThreshold"] != 25.0 || p["OverboughtThreshold"] != 70.0 {
		t.Errorf("Expected thresholds 25/70, got: %v/%v", p["OversoldThreshold"], p["OverboughtThreshold"])
	}

	strategy, err = NewTradeStrategyFactory("RSI", RSIParams{Period: 14, OverboughtThreshold: 80.0})
	if err != nil {
		t.Fatalf("Expected no error when OversoldThreshold is omitted, got: %v", err)
	}
	p = strategy.GetParams()
	if p["OversoldThreshold"] != 30.0 || p["OverboughtThreshold"] != 80.0 {
		t.Errorf("Expected thresholds 30/80, got: %v/%v", p["OversoldThreshold"], p["OverboughtThreshold"])
	}
}

func TestNewTradeStrategyFactory_RSI_OmittedOversoldAboveOverbought(t *testing.T) {
	// Oversold defaults to 30, which is not below an explicit overbought of 25
	_, err := NewTradeStrategyFactory("RSI", RSIParams{Period: 14, OverboughtThreshold: 25.0})
	if err == nil {
		t.Fatal("Expected error when default oversold is not below overbought, got nil")
	}
	expectedMessage := "OversoldThreshold must be less than OverboughtThreshold"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

func TestNewTradeStrategyFactory_RSI_OverboughtOutOfRange(t *testing.T) {
	_, err := NewTradeStrategyFactory("RSI", RSIParams{Period: 14, OversoldThreshold: 30.0, OverboughtThreshold: 150.0})
	if err == nil {
		t.Fatal("Expected error for out-of-range OverboughtThreshold, got nil")
	}
	expectedMessage := "OverboughtThreshold must be between 0 and 100"
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got: %s", expectedMessage, err.Error())
	}
}

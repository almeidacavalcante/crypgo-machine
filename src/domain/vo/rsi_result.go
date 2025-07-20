package vo

import (
	"fmt"
	"time"
)

type RSIResult struct {
	Value     float64
	Period    int
	Timestamp time.Time
	Signal    RSISignal
}

type RSISignal string

const (
	RSIOversold  RSISignal = "OVERSOLD"  // RSI < 30
	RSIOverbought RSISignal = "OVERBOUGHT" // RSI > 70
	RSINeutral   RSISignal = "NEUTRAL"   // 30 <= RSI <= 70
)

func NewRSIResult(value float64, period int, timestamp time.Time) (*RSIResult, error) {
	if value < 0 || value > 100 {
		return nil, fmt.Errorf("RSI value must be between 0 and 100, got: %.2f", value)
	}
	
	if period <= 0 {
		return nil, fmt.Errorf("RSI period must be positive, got: %d", period)
	}

	signal := determineRSISignal(value)

	return &RSIResult{
		Value:     value,
		Period:    period,
		Timestamp: timestamp,
		Signal:    signal,
	}, nil
}

func determineRSISignal(value float64) RSISignal {
	if value < 30 {
		return RSIOversold
	}
	if value > 70 {
		return RSIOverbought
	}
	return RSINeutral
}

func (r *RSIResult) IsOversold() bool {
	return r.Signal == RSIOversold
}

func (r *RSIResult) IsOverbought() bool {
	return r.Signal == RSIOverbought
}

func (r *RSIResult) IsNeutral() bool {
	return r.Signal == RSINeutral
}
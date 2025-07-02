package vo

import (
	"errors"
	"fmt"
)

type Drawdown struct {
	value       float64 // Percentage of maximum loss from peak
	maxValue    float64 // Peak value before drawdown
	minValue    float64 // Lowest value during drawdown
	duration    int     // Number of periods in drawdown
}

func NewDrawdown(maxValue, minValue float64, duration int) (Drawdown, error) {
	if maxValue <= 0 {
		return Drawdown{}, errors.New("max value must be positive")
	}
	
	if minValue < 0 {
		return Drawdown{}, errors.New("min value cannot be negative")
	}
	
	if minValue > maxValue {
		return Drawdown{}, errors.New("min value cannot be greater than max value")
	}
	
	if duration < 0 {
		return Drawdown{}, errors.New("duration cannot be negative")
	}
	
	// Calculate drawdown percentage
	value := ((maxValue - minValue) / maxValue) * 100
	
	return Drawdown{
		value:    value,
		maxValue: maxValue,
		minValue: minValue,
		duration: duration,
	}, nil
}

func (d Drawdown) GetValue() float64 {
	return d.value
}

func (d Drawdown) GetMaxValue() float64 {
	return d.maxValue
}

func (d Drawdown) GetMinValue() float64 {
	return d.minValue
}

func (d Drawdown) GetDuration() int {
	return d.duration
}

func (d Drawdown) IsAcceptable() bool {
	return d.value <= 20.0 // Less than 20% drawdown is acceptable
}

func (d Drawdown) IsLow() bool {
	return d.value <= 10.0 // Less than 10% is low risk
}

func (d Drawdown) IsHigh() bool {
	return d.value >= 30.0 // More than 30% is high risk
}

func (d Drawdown) String() string {
	return fmt.Sprintf("%.2f%% (%.2f -> %.2f over %d periods)", 
		d.value, d.maxValue, d.minValue, d.duration)
}
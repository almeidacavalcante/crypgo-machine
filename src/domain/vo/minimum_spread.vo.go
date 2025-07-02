package vo

import (
	"errors"
)

type MinimumSpread struct {
	value float64
}

func NewMinimumSpread(value float64) (MinimumSpread, error) {
	if value < 0 {
		return MinimumSpread{}, errors.New("minimum spread cannot be negative")
	}
	if value > 100 {
		return MinimumSpread{}, errors.New("minimum spread cannot exceed 100")
	}
	return MinimumSpread{value: value}, nil
}

func (m MinimumSpread) GetValue() float64 {
	return m.value
}

// HasSufficientSpread checks if the difference between fast and slow averages
// is greater than the minimum spread required to avoid whipsaw signals
func (m MinimumSpread) HasSufficientSpread(fast, slow float64) bool {
	if slow == 0 {
		return false
	}
	
	// Calculate percentage difference
	percentageDiff := ((fast - slow) / slow) * 100
	if percentageDiff < 0 {
		percentageDiff = -percentageDiff
	}
	
	return percentageDiff >= m.value
}
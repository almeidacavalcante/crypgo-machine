package vo

import (
	"errors"
	"fmt"
)

type WinRate struct {
	value float64 // Percentage from 0 to 100
}

func NewWinRate(wins, totalTrades int) (WinRate, error) {
	if totalTrades < 0 {
		return WinRate{}, errors.New("total trades cannot be negative")
	}
	
	if wins < 0 {
		return WinRate{}, errors.New("wins cannot be negative")
	}
	
	if wins > totalTrades {
		return WinRate{}, errors.New("wins cannot be greater than total trades")
	}
	
	if totalTrades == 0 {
		return WinRate{value: 0}, nil
	}
	
	value := (float64(wins) / float64(totalTrades)) * 100
	
	return WinRate{value: value}, nil
}

func NewWinRateFromPercentage(percentage float64) (WinRate, error) {
	if percentage < 0 || percentage > 100 {
		return WinRate{}, errors.New("win rate percentage must be between 0 and 100")
	}
	
	return WinRate{value: percentage}, nil
}

func (w WinRate) GetValue() float64 {
	return w.value
}

func (w WinRate) IsGood() bool {
	return w.value >= 50.0 // Above 50% is considered good
}

func (w WinRate) IsExcellent() bool {
	return w.value >= 70.0 // Above 70% is considered excellent
}

func (w WinRate) String() string {
	return fmt.Sprintf("%.2f%%", w.value)
}
package vo

import (
	"errors"
	"fmt"
)

type ProfitLoss struct {
	value    float64
	currency *Currency
}

func NewProfitLoss(value float64, currency *Currency) (ProfitLoss, error) {
	return ProfitLoss{
		value:    value,
		currency: currency,
	}, nil
}

func (p ProfitLoss) GetValue() float64 {
	return p.value
}

func (p ProfitLoss) GetCurrency() *Currency {
	return p.currency
}

func (p ProfitLoss) IsProfit() bool {
	return p.value > 0
}

func (p ProfitLoss) IsLoss() bool {
	return p.value < 0
}

func (p ProfitLoss) IsBreakEven() bool {
	return p.value == 0
}

func (p ProfitLoss) Add(other ProfitLoss) (ProfitLoss, error) {
	if p.currency.Code() != other.currency.Code() {
		return ProfitLoss{}, errors.New("cannot add profit/loss with different currencies")
	}
	
	return ProfitLoss{
		value:    p.value + other.value,
		currency: p.currency,
	}, nil
}

func (p ProfitLoss) Percentage(initialValue float64) float64 {
	if initialValue == 0 {
		return 0
	}
	return (p.value / initialValue) * 100
}

func (p ProfitLoss) String() string {
	sign := ""
	if p.value > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.2f %s", sign, p.value, p.currency.Code())
}
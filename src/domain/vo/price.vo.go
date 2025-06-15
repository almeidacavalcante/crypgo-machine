package vo

import "errors"

var (
	ErrInvalidPrice = errors.New("price cannot be negative")
)

type Price struct {
	amount   float64
	currency Currency
}

func NewPrice(amount float64, currency Currency) (*Price, error) {
	err := validatePrice(amount)
	if err != nil {
		return nil, err
	}
	return &Price{
		amount:   amount,
		currency: currency,
	}, nil
}

func validatePrice(amount float64) error {
	if amount < 0 {
		return ErrInvalidPrice
	}
	return nil
}

func (p *Price) Amount() float64 {
	return p.amount
}

func (p *Price) Currency() string {
	return p.currency.Code()
}

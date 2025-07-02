package vo

import "errors"

var (
	ErrInvalidPrice    = errors.New("price cannot be zero or negative")
	ErrInvalidCurrency = errors.New("currency should have length of 3")
)

type Price struct {
	amount   float64
	currency string
}

func NewPrice(amount float64, currency string) (Price, error) {
	err := validatePrice(amount, currency)
	if err != nil {
		return Price{}, err
	}
	return Price{
		amount:   amount,
		currency: currency,
	}, nil
}

func validatePrice(amount float64, currency string) error {
	if amount <= 0 {
		return ErrInvalidPrice
	}

	if len(currency) != 3 {
		return ErrInvalidCurrency
	}

	return nil
}

func (p *Price) GetAmount() float64 {
	return p.amount
}

func (p *Price) GetCurrency() string {
	return p.currency
}

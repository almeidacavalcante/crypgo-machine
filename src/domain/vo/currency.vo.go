package vo

import "errors"

var (
	ErrInvalidCurrency = errors.New("currency code must be a 3-letter string")
)

type Currency struct {
	code string
}

func NewCurrency(code string) (*Currency, error) {
	err := validateCurrency(code)
	if err != nil {
		return nil, err
	}
	return &Currency{
		code: code,
	}, nil
}

func validateCurrency(code string) error {
	if code == "" || len(code) != 3 {
		return ErrInvalidCurrency
	}
	return nil
}

func (c *Currency) Code() string {
	return c.code
}

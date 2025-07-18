package vo

import "fmt"

type Symbol struct {
	value string
}

func NewSymbol(val string) (Symbol, error) {
	// exemplos possível:
	// BTCBRL
	// SOLBRL

	err := validate(val)
	if err != nil {
		return Symbol{}, err
	}

	return Symbol{value: val}, nil
}

func (s Symbol) GetValue() string {
	return s.value
}

func validate(val string) error {
	allowedSymbols := map[string]struct{}{
		"BTCBRL": {},
		"SOLBRL": {},
		"ETHBRL": {},
	}

	if _, ok := allowedSymbols[val]; !ok {
		return fmt.Errorf("invalid symbol: %s", val)
	}

	return nil
}

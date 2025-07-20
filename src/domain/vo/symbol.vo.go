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
	// Verifica se tem pelo menos 6 caracteres (ex: BTCBRL)
	if len(val) < 6 {
		return fmt.Errorf("invalid symbol: %s (too short, minimum 6 characters)", val)
	}
	
	// Verifica se tem no máximo 15 caracteres
	if len(val) > 15 {
		return fmt.Errorf("invalid symbol: %s (too long, maximum 15 characters)", val)
	}

	// Verifica se contém apenas letras maiúsculas e números
	for _, char := range val {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("invalid symbol: %s (only uppercase letters and numbers allowed)", val)
		}
	}

	return nil
}

package vo

import "fmt"

type Symbol string

func NewSymbol(val string) (Symbol, error) {
	if _, ok := allowedSymbols[val]; !ok {
		return "", fmt.Errorf("invalid symbol: %s, allowed symbols: %s", val, getAllowedSymbols())
	}
	return Symbol(val), nil
}

var allowedSymbols = map[string]struct{}{
	"SOLBRL": {},
}

func getAllowedSymbols() []string {
	symbols := make([]string, 0, len(allowedSymbols))
	for symbol := range allowedSymbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

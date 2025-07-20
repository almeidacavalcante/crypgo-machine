package vo

import "testing"

func TestNewSymbol(t *testing.T) {
	btcbrl := "BTCBRL"

	s, err := NewSymbol(btcbrl)
	if err != nil {
		t.Errorf("exprected no error, but got %v", err)
	}
	if s.GetValue() != btcbrl {
		t.Errorf("expected to be %s", btcbrl)
	}
	if s == (Symbol{}) {
		t.Errorf("expected to be a symbol but got empty")
	}
}

func TestSymbol_Invalid(t *testing.T) {
	testCases := []struct {
		name   string
		symbol string
		errMsg string
	}{
		{"too short", "BTC", "too short"},
		{"too long", "BTCUSDTPERPETUALFUTURES", "too long"},
		{"lowercase", "btcusdt", "only uppercase"},
		{"special chars", "BTC-USDT", "only uppercase"},
		{"spaces", "BTC USDT", "only uppercase"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSymbol(tc.symbol)
			if err == nil {
				t.Errorf("expected error for %s", tc.symbol)
			}
			if s != (Symbol{}) {
				t.Errorf("expected empty symbol for %s", tc.symbol)
			}
		})
	}
}

func TestSymbol_Valid_BinancePatterns(t *testing.T) {
	validSymbols := []string{
		"BTCUSDT",
		"ETHBRL",
		"SOLBRL",
		"BNBBUSD",
		"ADAUSDT",
		"DOGEUSDT",
		"XRPUSDT",
		"DOTUSDT",
		"UNIUSDT",
		"LTCUSDT",
		"LINKUSDT",
		"MATICUSDT",
		"1000SHIBUSDT", // com números
		"1INCHUSDT",    // começando com número
	}

	for _, symbol := range validSymbols {
		t.Run(symbol, func(t *testing.T) {
			s, err := NewSymbol(symbol)
			if err != nil {
				t.Errorf("expected no error for %s, but got %v", symbol, err)
			}
			if s.GetValue() != symbol {
				t.Errorf("expected %s, got %s", symbol, s.GetValue())
			}
		})
	}
}

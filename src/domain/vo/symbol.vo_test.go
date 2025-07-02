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
	invalid := "INVALID"
	s, err := NewSymbol(invalid)
	if err == nil {
		t.Errorf("expected error for invalid symbol")
	}
	if s != (Symbol{}) {
		t.Errorf("expected to be empty symbol")
	}
}

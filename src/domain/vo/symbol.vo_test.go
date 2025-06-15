package vo

import (
	"testing"
)

func TestNewSymbol_ValidSymbol(t *testing.T) {
	s, err := NewSymbol("SOLBRL")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if s != "SOLBRL" {
		t.Errorf("expected symbol to be SOLBRL, got %v", s)
	}
}

func TestNewSymbol_InvalidSymbol(t *testing.T) {
	invalid := "INVALID"
	s, err := NewSymbol(invalid)
	if err == nil {
		t.Errorf("expected error for invalid symbol, got none")
	}
	if s != "" {
		t.Errorf("expected empty symbol on error, got %v", s)
	}
}

func TestGetAllowedSymbols(t *testing.T) {
	expected := "SOLBRL"
	found := false
	for _, s := range getAllowedSymbols() {
		if s == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find %v in allowed symbols", expected)
	}
}

package vo

import (
	"testing"
)

func TestNewKline_Valid(t *testing.T) {
	k, err := NewKline(10, 12, 15, 9, 100, 1720913950)
	if err != nil {
		t.Fatalf("expected valid kline, got error: %v", err)
	}
	if k.Open() != 10 || k.Close() != 12 || k.High() != 15 || k.Low() != 9 || k.Volume() != 100 || k.CloseTime() != 1720913950 {
		t.Fatal("field values not correctly set")
	}
}

func TestNewKline_NegativeOpen(t *testing.T) {
	_, err := NewKline(-1, 12, 15, 9, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for negative open, got nil")
	}
}

func TestNewKline_NegativeClose(t *testing.T) {
	_, err := NewKline(10, -2, 15, 9, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for negative close, got nil")
	}
}

func TestNewKline_NegativeHigh(t *testing.T) {
	_, err := NewKline(10, 12, -3, 9, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for negative high, got nil")
	}
}

func TestNewKline_NegativeLow(t *testing.T) {
	_, err := NewKline(10, 12, 15, -4, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for negative low, got nil")
	}
}

func TestNewKline_NegativeVolume(t *testing.T) {
	_, err := NewKline(10, 12, 15, 9, -5, 1720913950)
	if err == nil {
		t.Fatal("expected error for negative volume, got nil")
	}
}

func TestNewKline_ZeroOrNegativeCloseTime(t *testing.T) {
	_, err := NewKline(10, 12, 15, 9, 100, 0)
	if err == nil {
		t.Fatal("expected error for non-positive closeTime, got nil")
	}
	_, err = NewKline(10, 12, 15, 9, 100, -1)
	if err == nil {
		t.Fatal("expected error for non-positive closeTime, got nil")
	}
}

func TestNewKline_HighLessThanOthers(t *testing.T) {
	_, err := NewKline(10, 12, 8, 9, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for high < open/close/low, got nil")
	}
}

func TestNewKline_LowGreaterThanOthers(t *testing.T) {
	_, err := NewKline(10, 12, 15, 16, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for low > open/close/high, got nil")
	}
}

func TestNewKline_HighLessThanLow(t *testing.T) {
	_, err := NewKline(10, 12, 7, 9, 100, 1720913950)
	if err == nil {
		t.Fatal("expected error for high < low, got nil")
	}
}

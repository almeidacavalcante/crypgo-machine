package vo

import (
	"math"
	"testing"
	"time"
)

func TestNewFuturesBasis_ComputesPremiumAndAnnualizedBasis(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	expiry := now.Add(73 * 24 * time.Hour) // one fifth of a year

	basis, err := NewFuturesBasis("BTC-1FEB27", expiry, now, 60000, 60600, 0.0006)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertClose(t, "days to expiry", basis.GetDaysToExpiry(), 73)
	assertClose(t, "premium", basis.GetPremium(), 1.0)
	assertClose(t, "annualized", basis.GetAnnualized(), 5.0)
	assertClose(t, "net premium", basis.GetNetPremium(), 0.94)
	assertClose(t, "net annualized", basis.GetNetAnnualized(), 4.7)

	if basis.GetInstrument() != "BTC-1FEB27" {
		t.Errorf("expected instrument BTC-1FEB27, got %s", basis.GetInstrument())
	}
	if !basis.GetExpiry().Equal(expiry) {
		t.Errorf("expected expiry %v, got %v", expiry, basis.GetExpiry())
	}
	if basis.GetIndexPrice() != 60000 || basis.GetFuturesPrice() != 60600 {
		t.Errorf("unexpected prices: index %v, futures %v", basis.GetIndexPrice(), basis.GetFuturesPrice())
	}
}

func TestNewFuturesBasis_NegativeWhenFutureTradesBelowIndex(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	expiry := now.Add(73 * 24 * time.Hour)

	basis, err := NewFuturesBasis("BTC-1FEB27", expiry, now, 60000, 59400, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertClose(t, "premium", basis.GetPremium(), -1.0)
	assertClose(t, "annualized", basis.GetAnnualized(), -5.0)
}

func TestNewFuturesBasis_RejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)

	cases := []struct {
		name         string
		instrument   string
		expiry       time.Time
		indexPrice   float64
		futuresPrice float64
		feeRate      float64
	}{
		{"empty instrument", "", future, 60000, 60600, 0},
		{"zero index price", "BTC-1FEB27", future, 0, 60600, 0},
		{"negative index price", "BTC-1FEB27", future, -1, 60600, 0},
		{"zero futures price", "BTC-1FEB27", future, 60000, 0, 0},
		{"negative fee rate", "BTC-1FEB27", future, 60000, 60600, -0.001},
		{"expiry equal to now", "BTC-1FEB27", now, 60000, 60600, 0},
		{"expiry in the past", "BTC-1FEB27", now.Add(-time.Hour), 60000, 60600, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewFuturesBasis(tc.instrument, tc.expiry, now, tc.indexPrice, tc.futuresPrice, tc.feeRate); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s: expected %v, got %v", label, want, got)
	}
}

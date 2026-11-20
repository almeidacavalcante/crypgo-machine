package vo

import (
	"math"
	"testing"
)

func newTestSymbolFilter(t *testing.T, minQty, maxQty, stepSize float64) *SymbolFilter {
	t.Helper()
	sf, err := NewSymbolFilter("TESTPAIR", minQty, maxQty, stepSize, 0.001, 1000000, 0.001, 10)
	if err != nil {
		t.Fatalf("failed to create symbol filter: %v", err)
	}
	return sf
}

func TestSymbolFilter_AdjustQuantityToStepSize_RoundsToStepPrecision(t *testing.T) {
	testCases := []struct {
		name     string
		minQty   float64
		maxQty   float64
		stepSize float64
		quantity float64
		expected float64
		format   string
	}{
		// Floor(10.121457/0.1)*0.1 is 10.100000000000001 without precision rounding.
		{"step 0.1", 0.1, 90000, 0.1, 10.121457, 10.1, "10.1"},
		// Floor(1.156/0.01)*0.01 is 1.1500000000000001 without precision rounding.
		{"step 0.01", 0.01, 9000, 0.01, 1.156, 1.15, "1.15"},
		{"step 0.00001", 0.00001, 9000, 0.00001, 0.000259, 0.00025, "0.00025"},
		{"step 1", 1, 9000, 1, 7.9, 7, "7"},
		// 0.3/0.1 is 2.9999999999999996; an exact multiple must not lose a step.
		{"exact multiple kept", 0.1, 90000, 0.1, 0.3, 0.3, "0.3"},
		{"exact multiple 0.7 kept", 0.1, 90000, 0.1, 0.7, 0.7, "0.7"},
		{"clamped to maximum", 0.1, 50, 0.1, 75.55, 50, "50.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := newTestSymbolFilter(t, tc.minQty, tc.maxQty, tc.stepSize)

			adjusted := sf.AdjustQuantityToStepSize(tc.quantity)

			// Exact comparison on purpose: the adjusted value must be the
			// float64 closest to the decimal step multiple.
			if adjusted != tc.expected {
				t.Errorf("expected exactly %v, got %v", tc.expected, adjusted)
			}
			if adjusted > tc.quantity {
				t.Errorf("adjusted quantity %v exceeds requested quantity %v", adjusted, tc.quantity)
			}
			if err := sf.ValidateQuantity(adjusted); err != nil {
				t.Errorf("adjusted quantity should be valid: %v", err)
			}
			if got := sf.FormatQuantityString(adjusted); got != tc.format {
				t.Errorf("expected formatted %q, got %q", tc.format, got)
			}
		})
	}
}

func TestSymbolFilter_AdjustQuantityToStepSize_NeverExceedsRequestedQuantity(t *testing.T) {
	steps := []float64{1, 0.1, 0.01, 0.001, 0.0001, 0.00001, 0.000001, 0.00000001}

	for _, step := range steps {
		sf := newTestSymbolFilter(t, 0, 1000000, step)
		for i := 1; i <= 5000; i++ {
			// Quantities with up to 8 decimals, as returned by the exchange,
			// plus values carrying floating point noise from fee arithmetic.
			base := float64(i) * 0.00137
			candidates := []float64{
				base,
				math.Round(base*1e8) / 1e8,
				base * (1 - 0.001),
				math.Nextafter(base, 0),
			}
			for _, q := range candidates {
				adjusted := sf.AdjustQuantityToStepSize(q)
				if adjusted > q {
					t.Fatalf("step %v: adjusted %.17g exceeds requested %.17g", step, adjusted, q)
				}
				if q-adjusted >= step+1e-12 {
					t.Fatalf("step %v: adjusted %.17g dropped more than one step from %.17g", step, adjusted, q)
				}
			}
		}
	}
}

func TestSymbolFilter_AdjustQuantityToStepSize_DoesNotRoundUpToMinimum(t *testing.T) {
	sf := newTestSymbolFilter(t, 0.1, 90000, 0.1)

	adjusted := sf.AdjustQuantityToStepSize(0.05)

	if adjusted > 0.05 {
		t.Fatalf("quantity below minimum must not be rounded up, got %v", adjusted)
	}
	if err := sf.ValidateQuantity(adjusted); err == nil {
		t.Fatalf("expected adjusted quantity %v to be rejected as below minimum", adjusted)
	}
}

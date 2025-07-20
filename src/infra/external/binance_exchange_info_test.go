package external

import (
	"testing"
	"time"
)

func TestExchangeInfoService_GetSymbolFilters(t *testing.T) {
	// Create fake client and service
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	// Test getting XRPBRL filters (which should be in our fake data)
	filters, err := service.GetSymbolFilters("XRPBRL")
	if err != nil {
		t.Fatalf("Failed to get symbol filters: %v", err)
	}

	if filters == nil {
		t.Fatal("Symbol filters is nil")
	}

	if filters.Symbol != "XRPBRL" {
		t.Errorf("Expected symbol XRPBRL, got %s", filters.Symbol)
	}

	// Test LOT_SIZE filter
	if filters.LotSizeFilter == nil {
		t.Fatal("LOT_SIZE filter is nil")
	}

	expectedMinQty := 0.1
	if filters.LotSizeFilter.MinQty != expectedMinQty {
		t.Errorf("Expected MinQty %.1f, got %.8f", expectedMinQty, filters.LotSizeFilter.MinQty)
	}

	expectedStepSize := 0.1
	if filters.LotSizeFilter.StepSize != expectedStepSize {
		t.Errorf("Expected StepSize %.1f, got %.8f", expectedStepSize, filters.LotSizeFilter.StepSize)
	}

	// Test PRICE_FILTER
	if filters.PriceFilter == nil {
		t.Fatal("PRICE_FILTER is nil")
	}

	// Test MIN_NOTIONAL
	if filters.MinNotional == nil {
		t.Fatal("MIN_NOTIONAL filter is nil")
	}

	expectedMinNotional := 10.0
	if filters.MinNotional.MinNotional != expectedMinNotional {
		t.Errorf("Expected MinNotional %.1f, got %.8f", expectedMinNotional, filters.MinNotional.MinNotional)
	}

	t.Logf("✅ XRPBRL filters loaded correctly: MinQty=%.1f, StepSize=%.1f, MinNotional=%.1f", 
		filters.LotSizeFilter.MinQty, filters.LotSizeFilter.StepSize, filters.MinNotional.MinNotional)
}

func TestExchangeInfoService_QuantityValidation(t *testing.T) {
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	filters, err := service.GetSymbolFilters("XRPBRL")
	if err != nil {
		t.Fatalf("Failed to get symbol filters: %v", err)
	}

	testCases := []struct {
		name        string
		quantity    float64
		shouldError bool
		description string
	}{
		{
			name:        "Valid quantity - exact step",
			quantity:    10.1,
			shouldError: false,
			description: "10.1 should be valid (multiple of 0.1)",
		},
		{
			name:        "Valid quantity - minimum",
			quantity:    0.1,
			shouldError: false,
			description: "0.1 should be valid (minimum)",
		},
		{
			name:        "Invalid quantity - below minimum",
			quantity:    0.05,
			shouldError: true,
			description: "0.05 should be invalid (below minimum 0.1)",
		},
		{
			name:        "Invalid quantity - wrong step size",
			quantity:    10.121457,
			shouldError: true,
			description: "10.121457 should be invalid (not multiple of 0.1)",
		},
		{
			name:        "Valid quantity - large amount",
			quantity:    1000.0,
			shouldError: false,
			description: "1000.0 should be valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := filters.ValidateQuantity(tc.quantity)
			
			if tc.shouldError && err == nil {
				t.Errorf("Expected error for quantity %.6f, but got none. %s", tc.quantity, tc.description)
			}
			
			if !tc.shouldError && err != nil {
				t.Errorf("Expected no error for quantity %.6f, but got: %v. %s", tc.quantity, err, tc.description)
			}
			
			if err == nil {
				t.Logf("✅ %s: quantity %.6f is valid", tc.name, tc.quantity)
			} else {
				t.Logf("❌ %s: quantity %.6f failed validation: %v", tc.name, tc.quantity, err)
			}
		})
	}
}

func TestExchangeInfoService_QuantityAdjustment(t *testing.T) {
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	filters, err := service.GetSymbolFilters("XRPBRL")
	if err != nil {
		t.Fatalf("Failed to get symbol filters: %v", err)
	}

	testCases := []struct {
		name             string
		originalQuantity float64
		expectedAdjusted float64
		description      string
	}{
		{
			name:             "Round down to step",
			originalQuantity: 10.121457,
			expectedAdjusted: 10.1,
			description:      "Should round down 10.121457 to 10.1",
		},
		{
			name:             "Round down to step - small value",
			originalQuantity: 0.156,
			expectedAdjusted: 0.1,
			description:      "Should round down 0.156 to 0.1",
		},
		{
			name:             "Already valid - no change",
			originalQuantity: 15.0,
			expectedAdjusted: 15.0,
			description:      "15.0 should remain unchanged",
		},
		{
			name:             "Below minimum - adjust to minimum",
			originalQuantity: 0.05,
			expectedAdjusted: 0.1,
			description:      "Should adjust 0.05 to minimum 0.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjusted := filters.AdjustQuantityToStepSize(tc.originalQuantity)
			
			if adjusted != tc.expectedAdjusted {
				t.Errorf("Expected adjusted quantity %.1f, got %.8f. %s", tc.expectedAdjusted, adjusted, tc.description)
			} else {
				t.Logf("✅ %s: %.6f → %.1f", tc.name, tc.originalQuantity, adjusted)
			}
			
			// Verify the adjusted quantity is valid
			if err := filters.ValidateQuantity(adjusted); err != nil {
				t.Errorf("Adjusted quantity %.8f is still invalid: %v", adjusted, err)
			}
		})
	}
}

func TestExchangeInfoService_FormatQuantity(t *testing.T) {
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	testCases := []struct {
		symbol           string
		quantity         float64
		expectedFormat   string
		description      string
	}{
		{
			symbol:         "XRPBRL",
			quantity:       10.1,
			expectedFormat: "10.1",
			description:    "XRPBRL with 0.1 step size should format to 1 decimal",
		},
		{
			symbol:         "BTCUSDT",
			quantity:       0.00001,
			expectedFormat: "0.00001",
			description:    "BTCUSDT with 0.00001 step size should format to 5 decimals",
		},
		{
			symbol:         "SOLBRL",
			quantity:       1.25,
			expectedFormat: "1.25",
			description:    "SOLBRL with 0.01 step size should format to 2 decimals",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.symbol, func(t *testing.T) {
			filters, err := service.GetSymbolFilters(tc.symbol)
			if err != nil {
				t.Fatalf("Failed to get filters for %s: %v", tc.symbol, err)
			}

			formatted := filters.FormatQuantityForSymbol(tc.quantity)
			
			if formatted != tc.expectedFormat {
				t.Errorf("Expected format '%s', got '%s'. %s", tc.expectedFormat, formatted, tc.description)
			} else {
				t.Logf("✅ %s: %.8f → '%s'", tc.symbol, tc.quantity, formatted)
			}
		})
	}
}

func TestExchangeInfoService_NotionalValidation(t *testing.T) {
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	filters, err := service.GetSymbolFilters("XRPBRL")
	if err != nil {
		t.Fatalf("Failed to get symbol filters: %v", err)
	}

	testCases := []struct {
		name        string
		quantity    float64
		price       float64
		shouldError bool
		description string
	}{
		{
			name:        "Valid notional",
			quantity:    1.0,
			price:       20.0,
			shouldError: false,
			description: "1.0 * 20.0 = 20.0 BRL should be valid (> 10.0 minimum)",
		},
		{
			name:        "Invalid notional - too low",
			quantity:    0.1,
			price:       50.0,
			shouldError: true,
			description: "0.1 * 50.0 = 5.0 BRL should be invalid (< 10.0 minimum)",
		},
		{
			name:        "Exact minimum notional",
			quantity:    0.5,
			price:       20.0,
			shouldError: false,
			description: "0.5 * 20.0 = 10.0 BRL should be valid (= 10.0 minimum)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := filters.ValidateNotional(tc.quantity, tc.price)
			
			notionalValue := tc.quantity * tc.price
			
			if tc.shouldError && err == nil {
				t.Errorf("Expected error for notional %.2f, but got none. %s", notionalValue, tc.description)
			}
			
			if !tc.shouldError && err != nil {
				t.Errorf("Expected no error for notional %.2f, but got: %v. %s", notionalValue, err, tc.description)
			}
			
			if err == nil {
				t.Logf("✅ %s: notional %.2f BRL is valid", tc.name, notionalValue)
			} else {
				t.Logf("❌ %s: notional %.2f BRL failed: %v", tc.name, notionalValue, err)
			}
		})
	}
}

func TestExchangeInfoService_Cache(t *testing.T) {
	fakeClient := NewBinanceClientFake()
	service := NewExchangeInfoService(fakeClient)

	// First call should fetch from API
	start := time.Now()
	filters1, err := service.GetSymbolFilters("BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to get symbol filters: %v", err)
	}
	duration1 := time.Since(start)

	// Second call should use cache (should be faster)
	start = time.Now()
	filters2, err := service.GetSymbolFilters("BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to get symbol filters from cache: %v", err)
	}
	duration2 := time.Since(start)

	// Verify we got the same data
	if filters1.Symbol != filters2.Symbol {
		t.Errorf("Cache returned different symbol: %s vs %s", filters1.Symbol, filters2.Symbol)
	}

	// Cache should be faster (though this might not always be true in tests)
	t.Logf("First call: %v, Second call (cached): %v", duration1, duration2)

	// Check cache status
	status := service.GetCacheStatus()
	cachedCount, ok := status["cached_symbols_count"].(int)
	if !ok || cachedCount == 0 {
		t.Error("Cache should contain at least one symbol")
	}

	t.Logf("✅ Cache working: %d symbols cached", cachedCount)
}
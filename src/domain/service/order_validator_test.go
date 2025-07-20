package service

import (
	"crypgo-machine/src/infra/external"
	"testing"
)

func TestOrderValidatorService_ValidateOrder(t *testing.T) {
	// Setup
	fakeClient := external.NewBinanceClientFake()
	exchangeInfoService := external.NewExchangeInfoService(fakeClient)
	validator := NewOrderValidatorService(exchangeInfoService)

	testCases := []struct {
		name                string
		symbol              string
		quantity            float64
		price               float64
		expectValid         bool
		expectAdjustment    bool
		expectedAdjustedQty float64
		description         string
	}{
		{
			name:                "Valid XRPBRL order",
			symbol:              "XRPBRL",
			quantity:            10.1,
			price:               20.0,
			expectValid:         true,
			expectAdjustment:    false,
			expectedAdjustedQty: 10.1,
			description:         "Perfect valid order for XRPBRL",
		},
		{
			name:                "XRPBRL quantity needs adjustment",
			symbol:              "XRPBRL",
			quantity:            10.121457, // The problematic quantity from logs
			price:               19.76,
			expectValid:         true,
			expectAdjustment:    true,
			expectedAdjustedQty: 10.1,
			description:         "Should adjust 10.121457 down to 10.1 for XRPBRL",
		},
		{
			name:                "XRPBRL below minimum quantity",
			symbol:              "XRPBRL",
			quantity:            0.05,
			price:               20.0,
			expectValid:         true,
			expectAdjustment:    true,
			expectedAdjustedQty: 0.1,
			description:         "Should adjust 0.05 up to minimum 0.1",
		},
		{
			name:                "XRPBRL insufficient notional",
			symbol:              "XRPBRL",
			quantity:            0.1,
			price:               5.0, // 0.1 * 5.0 = 0.5 BRL < 10.0 minimum
			expectValid:         false,
			expectAdjustment:    false,
			expectedAdjustedQty: 0.1,
			description:         "Should fail due to insufficient notional value",
		},
		{
			name:                "BTCUSDT valid small quantity",
			symbol:              "BTCUSDT",
			quantity:            0.00001,
			price:               50000.0,
			expectValid:         true,
			expectAdjustment:    false,
			expectedAdjustedQty: 0.00001,
			description:         "Valid small BTC quantity",
		},
		{
			name:                "SOLBRL needs rounding",
			symbol:              "SOLBRL",
			quantity:            1.156,
			price:               1000.0,
			expectValid:         true,
			expectAdjustment:    true,
			expectedAdjustedQty: 1.15,
			description:         "Should round 1.156 down to 1.15 (step size 0.01)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validator.ValidateOrder(tc.symbol, tc.quantity, tc.price)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check validity
			if result.IsValid != tc.expectValid {
				t.Errorf("Expected IsValid=%t, got %t. Errors: %v", tc.expectValid, result.IsValid, result.ValidationErrors)
			}

			// Check adjustment
			wasAdjusted := result.AdjustedQuantity != result.OriginalQuantity
			if wasAdjusted != tc.expectAdjustment {
				t.Errorf("Expected adjustment=%t, got %t (%.8f → %.8f)", tc.expectAdjustment, wasAdjusted, result.OriginalQuantity, result.AdjustedQuantity)
			}

			// Check adjusted quantity value
			if tc.expectValid && result.AdjustedQuantity != tc.expectedAdjustedQty {
				t.Errorf("Expected adjusted quantity %.8f, got %.8f", tc.expectedAdjustedQty, result.AdjustedQuantity)
			}

			// Log results
			if result.IsValid {
				if wasAdjusted {
					t.Logf("✅ %s: %.8f → %.8f (%s)", tc.name, result.OriginalQuantity, result.AdjustedQuantity, result.FormattedQuantity)
				} else {
					t.Logf("✅ %s: %.8f valid (%s)", tc.name, result.OriginalQuantity, result.FormattedQuantity)
				}
			} else {
				t.Logf("❌ %s: %.8f failed - %v", tc.name, result.OriginalQuantity, result.ValidationErrors)
			}
		})
	}
}

func TestOrderValidatorService_ValidateAndAdjustQuantity(t *testing.T) {
	// Setup
	fakeClient := external.NewBinanceClientFake()
	exchangeInfoService := external.NewExchangeInfoService(fakeClient)
	validator := NewOrderValidatorService(exchangeInfoService)

	testCases := []struct {
		name            string
		symbol          string
		quantity        float64
		price           float64
		expectError     bool
		expectedQty     float64
		expectedFormat  string
		description     string
	}{
		{
			name:           "XRPBRL successful adjustment",
			symbol:         "XRPBRL",
			quantity:       10.121457,
			price:          19.76,
			expectError:    false,
			expectedQty:    10.1,
			expectedFormat: "10.1",
			description:    "Should successfully adjust and format XRPBRL quantity",
		},
		{
			name:           "BTCUSDT no adjustment needed",
			symbol:         "BTCUSDT",
			quantity:       0.00001,
			price:          50000.0,
			expectError:    false,
			expectedQty:    0.00001,
			expectedFormat: "0.00001",
			description:    "Should pass through valid BTCUSDT quantity",
		},
		{
			name:           "XRPBRL insufficient notional",
			symbol:         "XRPBRL",
			quantity:       0.1,
			price:          5.0,
			expectError:    true,
			expectedQty:    0.1,
			expectedFormat: "0.1",
			description:    "Should fail due to insufficient notional value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjustedQty, formattedQty, err := validator.ValidateAndAdjustQuantity(tc.symbol, tc.quantity, tc.price)

			// Check error expectation
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check results for successful cases
			if !tc.expectError {
				if adjustedQty != tc.expectedQty {
					t.Errorf("Expected adjusted quantity %.8f, got %.8f", tc.expectedQty, adjustedQty)
				}
				if formattedQty != tc.expectedFormat {
					t.Errorf("Expected formatted quantity '%s', got '%s'", tc.expectedFormat, formattedQty)
				}
				t.Logf("✅ %s: %.8f → %.8f ('%s')", tc.name, tc.quantity, adjustedQty, formattedQty)
			} else {
				t.Logf("❌ %s: %.8f failed as expected: %v", tc.name, tc.quantity, err)
			}
		})
	}
}

func TestOrderValidatorService_ValidateOrderBeforePlacement(t *testing.T) {
	// Setup
	fakeClient := external.NewBinanceClientFake()
	exchangeInfoService := external.NewExchangeInfoService(fakeClient)
	validator := NewOrderValidatorService(exchangeInfoService)

	testCases := []struct {
		name              string
		symbol            string
		quantity          float64
		price             float64
		expectProceed     bool
		expectWarnings    bool
		expectedAdjustedQty float64
		description       string
	}{
		{
			name:              "Valid order - should proceed",
			symbol:            "XRPBRL",
			quantity:          10.0,
			price:             20.0,
			expectProceed:     true,
			expectWarnings:    false,
			expectedAdjustedQty: 10.0,
			description:       "Perfect valid order should proceed without warnings",
		},
		{
			name:              "Needs adjustment - should proceed with warning",
			symbol:            "XRPBRL",
			quantity:          10.121457,
			price:             19.76,
			expectProceed:     true,
			expectWarnings:    true,
			expectedAdjustedQty: 10.1,
			description:       "Should proceed with quantity adjustment and warning",
		},
		{
			name:              "Invalid order - should not proceed",
			symbol:            "XRPBRL",
			quantity:          0.1,
			price:             5.0,
			expectProceed:     false,
			expectWarnings:    false,
			expectedAdjustedQty: 0.1,
			description:       "Invalid order should not proceed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adjustedQty, formattedQty, shouldProceed, warnings := validator.ValidateOrderBeforePlacement(tc.symbol, tc.quantity, tc.price)

			// Check proceed decision
			if shouldProceed != tc.expectProceed {
				t.Errorf("Expected shouldProceed=%t, got %t", tc.expectProceed, shouldProceed)
			}

			// Check warnings
			hasWarnings := len(warnings) > 0
			if hasWarnings != tc.expectWarnings {
				t.Errorf("Expected warnings=%t, got %t (warnings: %v)", tc.expectWarnings, hasWarnings, warnings)
			}

			// Check adjusted quantity for successful cases
			if shouldProceed && adjustedQty != tc.expectedAdjustedQty {
				t.Errorf("Expected adjusted quantity %.8f, got %.8f", tc.expectedAdjustedQty, adjustedQty)
			}

			// Log results
			if shouldProceed {
				warningStr := ""
				if hasWarnings {
					warningStr = " (with warnings)"
				}
				t.Logf("✅ %s: %.8f → %.8f ('%s') - PROCEED%s", tc.name, tc.quantity, adjustedQty, formattedQty, warningStr)
				if hasWarnings {
					for _, warning := range warnings {
						t.Logf("   ⚠️ %s", warning)
					}
				}
			} else {
				t.Logf("❌ %s: %.8f - DO NOT PROCEED", tc.name, tc.quantity)
			}
		})
	}
}

func TestOrderValidatorService_GetSymbolInfo(t *testing.T) {
	// Setup
	fakeClient := external.NewBinanceClientFake()
	exchangeInfoService := external.NewExchangeInfoService(fakeClient)
	validator := NewOrderValidatorService(exchangeInfoService)

	symbols := []string{"XRPBRL", "BTCUSDT", "ETHUSDT", "SOLBRL"}

	for _, symbol := range symbols {
		t.Run(symbol, func(t *testing.T) {
			info, err := validator.GetSymbolInfo(symbol)
			if err != nil {
				t.Fatalf("Failed to get symbol info for %s: %v", symbol, err)
			}

			// Check basic structure
			if info["symbol"] != symbol {
				t.Errorf("Expected symbol %s, got %v", symbol, info["symbol"])
			}

			// Check for required filters
			if _, exists := info["lot_size"]; !exists {
				t.Error("Missing lot_size filter")
			}

			if _, exists := info["price_filter"]; !exists {
				t.Error("Missing price_filter")
			}

			if _, exists := info["min_notional"]; !exists {
				t.Error("Missing min_notional filter")
			}

			t.Logf("✅ %s symbol info retrieved successfully", symbol)
			
			// Log some details for verification
			if lotSize, ok := info["lot_size"].(map[string]interface{}); ok {
				t.Logf("   Lot Size: min=%.8f, max=%.0f, step=%.8f", 
					lotSize["min_qty"], lotSize["max_qty"], lotSize["step_size"])
			}
			
			if minNotional, ok := info["min_notional"].(float64); ok {
				t.Logf("   Min Notional: %.2f", minNotional)
			}
		})
	}
}

// Integration test simulating the exact scenario from production logs
func TestOrderValidatorService_ProductionScenario(t *testing.T) {
	// Setup
	fakeClient := external.NewBinanceClientFake()
	exchangeInfoService := external.NewExchangeInfoService(fakeClient)
	validator := NewOrderValidatorService(exchangeInfoService)

	// Simulate the exact scenario from production logs
	symbol := "XRPBRL"
	problematicQuantity := 10.121457
	price := 19.76

	t.Logf("🔍 Testing production scenario: %s quantity %.6f at price %.2f", symbol, problematicQuantity, price)

	// Test the validation
	result, err := validator.ValidateOrder(symbol, problematicQuantity, price)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.IsValid {
		t.Fatalf("Order should be valid after adjustment, errors: %v", result.ValidationErrors)
	}

	if result.AdjustedQuantity == result.OriginalQuantity {
		t.Error("Expected quantity to be adjusted")
	}

	// The adjusted quantity should be valid for XRPBRL (step size 0.1)
	expectedAdjusted := 10.1
	tolerance := 1e-6
	if result.AdjustedQuantity < expectedAdjusted - tolerance || result.AdjustedQuantity > expectedAdjusted + tolerance {
		t.Errorf("Expected adjusted quantity %.1f, got %.8f", expectedAdjusted, result.AdjustedQuantity)
	}

	// Test the formatting
	if result.FormattedQuantity != "10.1" {
		t.Errorf("Expected formatted quantity '10.1', got '%s'", result.FormattedQuantity)
	}

	// Test the complete validation flow
	adjustedQty, formattedQty, shouldProceed, warnings := validator.ValidateOrderBeforePlacement(symbol, problematicQuantity, price)

	if !shouldProceed {
		t.Error("Order should proceed after adjustment")
	}

	if len(warnings) == 0 {
		t.Error("Expected warnings about quantity adjustment")
	}

	if adjustedQty < 10.1 - tolerance || adjustedQty > 10.1 + tolerance {
		t.Errorf("Expected final adjusted quantity 10.1, got %.8f", adjustedQty)
	}

	if formattedQty != "10.1" {
		t.Errorf("Expected final formatted quantity '10.1', got '%s'", formattedQty)
	}

	t.Logf("✅ Production scenario resolved successfully:")
	t.Logf("   Original: %.6f", problematicQuantity)
	t.Logf("   Adjusted: %.1f", adjustedQty)
	t.Logf("   Formatted: '%s'", formattedQty)
	t.Logf("   Should proceed: %t", shouldProceed)
	t.Logf("   Warnings: %v", warnings)

	// This should now work with Binance API instead of causing LOT_SIZE error
}
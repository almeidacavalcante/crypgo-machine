package vo

import (
	"fmt"
	"math"
)

// SymbolFilter contains trading rules for symbol validation
type SymbolFilter struct {
	symbol        string
	minQuantity   float64
	maxQuantity   float64
	stepSize      float64
	minPrice      float64
	maxPrice      float64
	tickSize      float64
	minNotional   float64
}

// NewSymbolFilter creates a new SymbolFilter
func NewSymbolFilter(
	symbol string,
	minQuantity, maxQuantity, stepSize float64,
	minPrice, maxPrice, tickSize float64,
	minNotional float64,
) (*SymbolFilter, error) {
	
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}
	
	if minQuantity < 0 || maxQuantity < 0 || stepSize <= 0 {
		return nil, fmt.Errorf("invalid quantity constraints for %s", symbol)
	}
	
	if minQuantity > maxQuantity {
		return nil, fmt.Errorf("minQuantity cannot be greater than maxQuantity for %s", symbol)
	}
	
	if minPrice < 0 || maxPrice < 0 || tickSize <= 0 {
		return nil, fmt.Errorf("invalid price constraints for %s", symbol)
	}
	
	if minNotional < 0 {
		return nil, fmt.Errorf("invalid notional constraint for %s", symbol)
	}

	return &SymbolFilter{
		symbol:        symbol,
		minQuantity:   minQuantity,
		maxQuantity:   maxQuantity,
		stepSize:      stepSize,
		minPrice:      minPrice,
		maxPrice:      maxPrice,
		tickSize:      tickSize,
		minNotional:   minNotional,
	}, nil
}

// GetSymbol returns the symbol
func (sf *SymbolFilter) GetSymbol() string {
	return sf.symbol
}

// GetMinQuantity returns the minimum quantity
func (sf *SymbolFilter) GetMinQuantity() float64 {
	return sf.minQuantity
}

// GetMaxQuantity returns the maximum quantity
func (sf *SymbolFilter) GetMaxQuantity() float64 {
	return sf.maxQuantity
}

// GetStepSize returns the step size for quantity
func (sf *SymbolFilter) GetStepSize() float64 {
	return sf.stepSize
}

// GetMinPrice returns the minimum price
func (sf *SymbolFilter) GetMinPrice() float64 {
	return sf.minPrice
}

// GetMaxPrice returns the maximum price
func (sf *SymbolFilter) GetMaxPrice() float64 {
	return sf.maxPrice
}

// GetTickSize returns the tick size for price
func (sf *SymbolFilter) GetTickSize() float64 {
	return sf.tickSize
}

// GetMinNotional returns the minimum notional value
func (sf *SymbolFilter) GetMinNotional() float64 {
	return sf.minNotional
}

// ValidateQuantity validates if quantity meets the symbol requirements
func (sf *SymbolFilter) ValidateQuantity(quantity float64) error {
	if quantity < sf.minQuantity {
		return fmt.Errorf("quantity %.8f is below minimum %.8f for %s", quantity, sf.minQuantity, sf.symbol)
	}
	
	if quantity > sf.maxQuantity {
		return fmt.Errorf("quantity %.8f exceeds maximum %.8f for %s", quantity, sf.maxQuantity, sf.symbol)
	}

	// Check step size compliance
	if sf.stepSize > 0 {
		// Calculate how many steps the quantity represents
		steps := quantity / sf.stepSize
		// Check if it's a whole number of steps (with small tolerance for floating point precision)
		remainder := math.Abs(steps - math.Round(steps))
		if remainder > 1e-8 {
			return fmt.Errorf("quantity %.8f does not comply with step size %.8f for %s", quantity, sf.stepSize, sf.symbol)
		}
	}

	return nil
}

// AdjustQuantityToStepSize adjusts quantity to comply with step size
func (sf *SymbolFilter) AdjustQuantityToStepSize(quantity float64) float64 {
	if sf.stepSize <= 0 {
		return quantity
	}

	// Round down to the nearest step
	steps := math.Floor(quantity / sf.stepSize)
	adjustedQuantity := steps * sf.stepSize

	// Ensure we don't go below minimum
	if adjustedQuantity < sf.minQuantity {
		// Try rounding up instead
		steps = math.Ceil(quantity / sf.stepSize)
		adjustedQuantity = steps * sf.stepSize
		
		// If still below minimum or above maximum, return the minimum
		if adjustedQuantity < sf.minQuantity || adjustedQuantity > sf.maxQuantity {
			adjustedQuantity = sf.minQuantity
		}
	}

	// Ensure we don't exceed maximum
	if adjustedQuantity > sf.maxQuantity {
		// Round down to fit within maximum
		steps = math.Floor(sf.maxQuantity / sf.stepSize)
		adjustedQuantity = steps * sf.stepSize
	}

	return adjustedQuantity
}

// FormatQuantityString formats quantity with appropriate precision based on step size
func (sf *SymbolFilter) FormatQuantityString(quantity float64) string {
	if sf.stepSize <= 0 {
		return fmt.Sprintf("%.6f", quantity) // Default fallback
	}

	// Determine decimal places based on step size
	decimalPlaces := sf.calculateDecimalPlaces(sf.stepSize)
	
	// Limit to reasonable precision
	if decimalPlaces > 8 {
		decimalPlaces = 8
	}

	formatStr := fmt.Sprintf("%%.%df", decimalPlaces)
	return fmt.Sprintf(formatStr, quantity)
}

// calculateDecimalPlaces determines how many decimal places are needed for step size
func (sf *SymbolFilter) calculateDecimalPlaces(stepSize float64) int {
	if stepSize >= 1.0 {
		return 0
	}
	
	// Convert to string and count decimal places
	stepStr := fmt.Sprintf("%.8f", stepSize)
	
	// Find the decimal point
	decimalIndex := -1
	for i, char := range stepStr {
		if char == '.' {
			decimalIndex = i
			break
		}
	}
	
	if decimalIndex == -1 {
		return 0
	}
	
	// Count significant decimal places (ignore trailing zeros)
	decimalPlaces := 0
	for i := len(stepStr) - 1; i > decimalIndex; i-- {
		if stepStr[i] != '0' {
			decimalPlaces = i - decimalIndex
			break
		}
	}
	
	return decimalPlaces
}

// ValidatePrice validates if price meets the symbol requirements
func (sf *SymbolFilter) ValidatePrice(price float64) error {
	if price < sf.minPrice {
		return fmt.Errorf("price %.8f is below minimum %.8f for %s", price, sf.minPrice, sf.symbol)
	}
	
	if price > sf.maxPrice {
		return fmt.Errorf("price %.8f exceeds maximum %.8f for %s", price, sf.maxPrice, sf.symbol)
	}

	// Check tick size compliance
	if sf.tickSize > 0 {
		steps := price / sf.tickSize
		remainder := math.Abs(steps - math.Round(steps))
		if remainder > 1e-8 {
			return fmt.Errorf("price %.8f does not comply with tick size %.8f for %s", price, sf.tickSize, sf.symbol)
		}
	}

	return nil
}

// ValidateNotional validates if order value meets minimum notional requirements
func (sf *SymbolFilter) ValidateNotional(quantity, price float64) error {
	if sf.minNotional <= 0 {
		return nil // No notional requirement
	}

	notionalValue := quantity * price
	if notionalValue < sf.minNotional {
		return fmt.Errorf("notional value %.2f is below minimum %.2f for %s", notionalValue, sf.minNotional, sf.symbol)
	}

	return nil
}

// ValidateOrder performs complete order validation
func (sf *SymbolFilter) ValidateOrder(quantity, price float64) error {
	if err := sf.ValidateQuantity(quantity); err != nil {
		return err
	}
	
	if err := sf.ValidatePrice(price); err != nil {
		return err
	}
	
	if err := sf.ValidateNotional(quantity, price); err != nil {
		return err
	}
	
	return nil
}

// String returns a string representation of the filter
func (sf *SymbolFilter) String() string {
	return fmt.Sprintf("SymbolFilter{%s: qty[%.8f-%.8f/%.8f], price[%.8f-%.8f/%.8f], notional:%.2f}",
		sf.symbol,
		sf.minQuantity, sf.maxQuantity, sf.stepSize,
		sf.minPrice, sf.maxPrice, sf.tickSize,
		sf.minNotional,
	)
}
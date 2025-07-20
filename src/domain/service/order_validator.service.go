package service

import (
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
)

// OrderValidationResult contains the result of order validation
type OrderValidationResult struct {
	IsValid           bool
	OriginalQuantity  float64
	AdjustedQuantity  float64
	FormattedQuantity string
	ValidationErrors  []string
	Warnings          []string
	SymbolFilters     *vo.SymbolFilter
}

// OrderValidatorService validates orders against Binance symbol constraints
type OrderValidatorService struct {
	exchangeInfoService *external.ExchangeInfoService
}

// NewOrderValidatorService creates a new order validator
func NewOrderValidatorService(exchangeInfoService *external.ExchangeInfoService) *OrderValidatorService {
	return &OrderValidatorService{
		exchangeInfoService: exchangeInfoService,
	}
}

// ValidateOrder validates an order and provides corrections if needed
func (s *OrderValidatorService) ValidateOrder(symbol string, quantity, price float64) (*OrderValidationResult, error) {
	result := &OrderValidationResult{
		OriginalQuantity:  quantity,
		AdjustedQuantity:  quantity,
		ValidationErrors:  make([]string, 0),
		Warnings:          make([]string, 0),
	}

	// Get symbol filters from exchange info
	symbolFilters, err := s.exchangeInfoService.GetSymbolFilters(symbol)
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Failed to get exchange info for %s: %v", symbol, err))
		result.IsValid = false
		result.FormattedQuantity = fmt.Sprintf("%.6f", quantity) // Fallback formatting
		return result, nil
	}

	// Convert exchange info filters to domain value object
	domainFilter, err := s.convertToSymbolFilter(symbolFilters)
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Failed to process filters for %s: %v", symbol, err))
		result.IsValid = false
		result.FormattedQuantity = fmt.Sprintf("%.6f", quantity)
		return result, nil
	}

	result.SymbolFilters = domainFilter

	// Validate original quantity
	quantityErr := domainFilter.ValidateQuantity(quantity)
	priceErr := domainFilter.ValidatePrice(price)
	notionalErr := domainFilter.ValidateNotional(quantity, price)

	// If quantity fails step size validation, try to adjust it
	if quantityErr != nil {
		adjustedQuantity := domainFilter.AdjustQuantityToStepSize(quantity)
		result.AdjustedQuantity = adjustedQuantity
		
		// Re-validate the adjusted quantity
		adjustedQuantityErr := domainFilter.ValidateQuantity(adjustedQuantity)
		adjustedNotionalErr := domainFilter.ValidateNotional(adjustedQuantity, price)
		
		if adjustedQuantityErr == nil && adjustedNotionalErr == nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Quantity adjusted from %.8f to %.8f to comply with step size %.8f", 
				quantity, adjustedQuantity, domainFilter.GetStepSize()))
			result.IsValid = true
		} else {
			result.ValidationErrors = append(result.ValidationErrors, quantityErr.Error())
			if adjustedQuantityErr != nil {
				result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Even after adjustment: %v", adjustedQuantityErr))
			}
			if adjustedNotionalErr != nil {
				result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Adjusted quantity notional error: %v", adjustedNotionalErr))
			}
			result.IsValid = false
		}
	} else {
		result.IsValid = true
	}

	// Add price validation errors
	if priceErr != nil {
		result.ValidationErrors = append(result.ValidationErrors, priceErr.Error())
		result.IsValid = false
	}

	// Add notional validation errors (for original quantity if no adjustment was made)
	if quantityErr == nil && notionalErr != nil {
		result.ValidationErrors = append(result.ValidationErrors, notionalErr.Error())
		result.IsValid = false
	}

	// Format the final quantity
	result.FormattedQuantity = domainFilter.FormatQuantityString(result.AdjustedQuantity)

	return result, nil
}

// ValidateAndAdjustQuantity performs validation and returns the best quantity to use
func (s *OrderValidatorService) ValidateAndAdjustQuantity(symbol string, quantity, price float64) (float64, string, error) {
	result, err := s.ValidateOrder(symbol, quantity, price)
	if err != nil {
		return quantity, fmt.Sprintf("%.6f", quantity), fmt.Errorf("validation failed: %v", err)
	}

	if !result.IsValid {
		errorMsg := fmt.Sprintf("Order validation failed for %s: %v", symbol, result.ValidationErrors)
		return quantity, fmt.Sprintf("%.6f", quantity), fmt.Errorf(errorMsg)
	}

	return result.AdjustedQuantity, result.FormattedQuantity, nil
}

// convertToSymbolFilter converts external symbol filters to domain value object
func (s *OrderValidatorService) convertToSymbolFilter(filters *external.SymbolFilters) (*vo.SymbolFilter, error) {
	if filters == nil {
		return nil, fmt.Errorf("symbol filters is nil")
	}

	// Extract values with defaults
	var minQty, maxQty, stepSize float64 = 0, 999999999, 1
	var minPrice, maxPrice, tickSize float64 = 0, 999999999, 0.01
	var minNotional float64 = 0

	if filters.LotSizeFilter != nil {
		minQty = filters.LotSizeFilter.MinQty
		maxQty = filters.LotSizeFilter.MaxQty
		stepSize = filters.LotSizeFilter.StepSize
	}

	if filters.PriceFilter != nil {
		minPrice = filters.PriceFilter.MinPrice
		maxPrice = filters.PriceFilter.MaxPrice
		tickSize = filters.PriceFilter.TickSize
	}

	if filters.MinNotional != nil {
		minNotional = filters.MinNotional.MinNotional
	}

	return vo.NewSymbolFilter(
		filters.Symbol,
		minQty, maxQty, stepSize,
		minPrice, maxPrice, tickSize,
		minNotional,
	)
}

// GetSymbolInfo returns symbol information for debugging
func (s *OrderValidatorService) GetSymbolInfo(symbol string) (map[string]interface{}, error) {
	filters, err := s.exchangeInfoService.GetSymbolFilters(symbol)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"symbol": filters.Symbol,
		"last_updated": filters.LastUpdated,
	}

	if filters.LotSizeFilter != nil {
		info["lot_size"] = map[string]interface{}{
			"min_qty":   filters.LotSizeFilter.MinQty,
			"max_qty":   filters.LotSizeFilter.MaxQty,
			"step_size": filters.LotSizeFilter.StepSize,
		}
	}

	if filters.PriceFilter != nil {
		info["price_filter"] = map[string]interface{}{
			"min_price": filters.PriceFilter.MinPrice,
			"max_price": filters.PriceFilter.MaxPrice,
			"tick_size": filters.PriceFilter.TickSize,
		}
	}

	if filters.MinNotional != nil {
		info["min_notional"] = filters.MinNotional.MinNotional
	}

	return info, nil
}

// ValidateOrderBeforePlacement is a convenience method that validates and logs
func (s *OrderValidatorService) ValidateOrderBeforePlacement(symbol string, quantity, price float64) (adjustedQty float64, formattedQty string, shouldProceed bool, warnings []string) {
	result, err := s.ValidateOrder(symbol, quantity, price)
	if err != nil {
		fmt.Printf("❌ Order validation error for %s: %v\n", symbol, err)
		return quantity, fmt.Sprintf("%.6f", quantity), false, []string{err.Error()}
	}

	if !result.IsValid {
		fmt.Printf("❌ Order validation failed for %s:\n", symbol)
		for _, errMsg := range result.ValidationErrors {
			fmt.Printf("   • %s\n", errMsg)
		}
		return quantity, fmt.Sprintf("%.6f", quantity), false, result.ValidationErrors
	}

	// Log warnings if quantity was adjusted
	if len(result.Warnings) > 0 {
		fmt.Printf("⚠️ Order adjustments for %s:\n", symbol)
		for _, warning := range result.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
	}

	if result.AdjustedQuantity != result.OriginalQuantity {
		fmt.Printf("📐 Quantity adjusted for %s: %.8f → %.8f (formatted: %s)\n", 
			symbol, result.OriginalQuantity, result.AdjustedQuantity, result.FormattedQuantity)
	}

	return result.AdjustedQuantity, result.FormattedQuantity, true, result.Warnings
}
package external

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
)

// SymbolFilters contains trading rules for a specific symbol
type SymbolFilters struct {
	Symbol          string
	LotSizeFilter   *LotSizeFilter
	PriceFilter     *PriceFilter
	MinNotional     *MinNotionalFilter
	LastUpdated     time.Time
}

// LotSizeFilter contains quantity constraints
type LotSizeFilter struct {
	MinQty   float64
	MaxQty   float64
	StepSize float64
}

// PriceFilter contains price constraints
type PriceFilter struct {
	MinPrice float64
	MaxPrice float64
	TickSize float64
}

// MinNotionalFilter contains minimum notional value constraint
type MinNotionalFilter struct {
	MinNotional float64
}

// ExchangeInfoService manages symbol trading rules from Binance
type ExchangeInfoService struct {
	client         BinanceClientInterface
	symbolFilters  map[string]*SymbolFilters
	mu             sync.RWMutex
	cacheTimeout   time.Duration
	lastFullUpdate time.Time
}

// NewExchangeInfoService creates a new exchange info service
func NewExchangeInfoService(client BinanceClientInterface) *ExchangeInfoService {
	return &ExchangeInfoService{
		client:        client,
		symbolFilters: make(map[string]*SymbolFilters),
		cacheTimeout:  30 * time.Minute, // Cache for 30 minutes
	}
}

// GetSymbolFilters returns trading rules for a specific symbol
func (s *ExchangeInfoService) GetSymbolFilters(symbol string) (*SymbolFilters, error) {
	s.mu.RLock()
	filters, exists := s.symbolFilters[symbol]
	s.mu.RUnlock()

	// Check if we have cached data that's still valid
	if exists && time.Since(filters.LastUpdated) < s.cacheTimeout {
		return filters, nil
	}

	// If cache is expired or missing, refresh from API
	err := s.refreshSymbolInfo(symbol)
	if err != nil {
		// If refresh fails but we have cached data, return it with warning
		if exists {
			fmt.Printf("⚠️ Failed to refresh exchange info for %s, using cached data: %v\n", symbol, err)
			return filters, nil
		}
		return nil, fmt.Errorf("failed to get exchange info for %s: %v", symbol, err)
	}

	s.mu.RLock()
	filters = s.symbolFilters[symbol]
	s.mu.RUnlock()

	return filters, nil
}

// refreshSymbolInfo updates symbol information from Binance API
func (s *ExchangeInfoService) refreshSymbolInfo(symbol string) error {
	exchangeInfo, err := s.client.NewGetExchangeInfoService().Do(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get exchange info: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the specific symbol or update all if doing full refresh
	symbolFound := false
	for _, symbolInfo := range exchangeInfo.Symbols {
		if symbolInfo.Symbol == symbol || symbol == "" {
			filters := s.parseSymbolFilters(&symbolInfo)
			s.symbolFilters[symbolInfo.Symbol] = filters
			if symbolInfo.Symbol == symbol {
				symbolFound = true
			}
		}
	}

	if symbol != "" && !symbolFound {
		return fmt.Errorf("symbol %s not found in exchange info", symbol)
	}

	s.lastFullUpdate = time.Now()
	return nil
}

// parseSymbolFilters extracts trading rules from Binance symbol info
func (s *ExchangeInfoService) parseSymbolFilters(symbolInfo *binance.Symbol) *SymbolFilters {
	filters := &SymbolFilters{
		Symbol:      symbolInfo.Symbol,
		LastUpdated: time.Now(),
	}

	for _, filter := range symbolInfo.Filters {
		switch filter["filterType"] {
		case "LOT_SIZE":
			filters.LotSizeFilter = &LotSizeFilter{
				MinQty:   s.parseFloat(filter["minQty"]),
				MaxQty:   s.parseFloat(filter["maxQty"]),
				StepSize: s.parseFloat(filter["stepSize"]),
			}
		case "PRICE_FILTER":
			filters.PriceFilter = &PriceFilter{
				MinPrice: s.parseFloat(filter["minPrice"]),
				MaxPrice: s.parseFloat(filter["maxPrice"]),
				TickSize: s.parseFloat(filter["tickSize"]),
			}
		case "MIN_NOTIONAL":
			filters.MinNotional = &MinNotionalFilter{
				MinNotional: s.parseFloat(filter["minNotional"]),
			}
		case "NOTIONAL":
			// Some symbols use NOTIONAL instead of MIN_NOTIONAL
			if filters.MinNotional == nil {
				filters.MinNotional = &MinNotionalFilter{
					MinNotional: s.parseFloat(filter["minNotional"]),
				}
			}
		}
	}

	return filters
}

// parseFloat safely converts string to float64
func (s *ExchangeInfoService) parseFloat(value interface{}) float64 {
	if value == nil {
		return 0
	}
	
	str, ok := value.(string)
	if !ok {
		return 0
	}
	
	result, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	
	return result
}

// ValidateQuantity checks if quantity is valid for the symbol
func (filters *SymbolFilters) ValidateQuantity(quantity float64) error {
	if filters.LotSizeFilter == nil {
		return fmt.Errorf("no LOT_SIZE filter found for symbol %s", filters.Symbol)
	}

	lot := filters.LotSizeFilter
	
	if quantity < lot.MinQty {
		return fmt.Errorf("quantity %.8f is below minimum %.8f for %s", quantity, lot.MinQty, filters.Symbol)
	}
	
	if quantity > lot.MaxQty {
		return fmt.Errorf("quantity %.8f exceeds maximum %.8f for %s", quantity, lot.MaxQty, filters.Symbol)
	}

	// Check step size compliance with tolerance for floating point precision
	if lot.StepSize > 0 {
		steps := quantity / lot.StepSize
		roundedSteps := float64(int64(steps + 0.5)) // Round to nearest integer
		tolerance := 1e-8
		
		if steps < roundedSteps - tolerance || steps > roundedSteps + tolerance {
			return fmt.Errorf("quantity %.8f does not comply with step size %.8f for %s", quantity, lot.StepSize, filters.Symbol)
		}
	}

	return nil
}

// AdjustQuantityToStepSize adjusts quantity to comply with step size
func (filters *SymbolFilters) AdjustQuantityToStepSize(quantity float64) float64 {
	if filters.LotSizeFilter == nil || filters.LotSizeFilter.StepSize <= 0 {
		return quantity
	}

	stepSize := filters.LotSizeFilter.StepSize
	steps := quantity / stepSize
	adjustedSteps := float64(int64(steps)) // Floor to nearest step
	adjustedQuantity := adjustedSteps * stepSize

	// Round to avoid floating point precision issues
	adjustedQuantity = roundToStepSizePrecision(adjustedQuantity, stepSize)

	// Ensure we don't go below minimum
	if adjustedQuantity < filters.LotSizeFilter.MinQty {
		adjustedQuantity = filters.LotSizeFilter.MinQty
	}

	return adjustedQuantity
}

// roundToStepSizePrecision rounds value to the precision of step size
func roundToStepSizePrecision(value, stepSize float64) float64 {
	// Determine decimal places based on step size
	stepStr := fmt.Sprintf("%.10f", stepSize)
	decimalPlaces := 0
	
	dotIndex := -1
	for i, char := range stepStr {
		if char == '.' {
			dotIndex = i
			break
		}
	}
	
	if dotIndex != -1 {
		// Count significant decimal places (ignore trailing zeros)
		for i := len(stepStr) - 1; i > dotIndex; i-- {
			if stepStr[i] != '0' {
				decimalPlaces = i - dotIndex
				break
			}
		}
	}
	
	// Round to the determined precision
	multiplier := float64(1)
	for i := 0; i < decimalPlaces; i++ {
		multiplier *= 10
	}
	
	return float64(int64(value*multiplier + 0.5)) / multiplier
}

// FormatQuantityForSymbol formats quantity with appropriate precision
func (filters *SymbolFilters) FormatQuantityForSymbol(quantity float64) string {
	if filters.LotSizeFilter == nil {
		return fmt.Sprintf("%.6f", quantity) // Default fallback
	}

	stepSize := filters.LotSizeFilter.StepSize
	if stepSize <= 0 {
		return fmt.Sprintf("%.6f", quantity)
	}

	// Determine decimal places based on step size
	decimalPlaces := calculateDecimalPlaces(stepSize)
	
	// Limit to reasonable precision
	if decimalPlaces > 8 {
		decimalPlaces = 8
	}

	formatStr := fmt.Sprintf("%%.%df", decimalPlaces)
	return fmt.Sprintf(formatStr, quantity)
}

// calculateDecimalPlaces determines how many decimal places are needed for step size
func calculateDecimalPlaces(stepSize float64) int {
	if stepSize >= 1.0 {
		return 0
	}
	
	// Convert to string and count decimal places
	stepStr := fmt.Sprintf("%.10f", stepSize)
	
	// Find the decimal point
	dotIndex := -1
	for i, char := range stepStr {
		if char == '.' {
			dotIndex = i
			break
		}
	}
	
	if dotIndex == -1 {
		return 0
	}
	
	// Count significant decimal places (ignore trailing zeros)
	decimalPlaces := 0
	for i := len(stepStr) - 1; i > dotIndex; i-- {
		if stepStr[i] != '0' {
			decimalPlaces = i - dotIndex
			break
		}
	}
	
	return decimalPlaces
}

// ValidateNotional checks if order value meets minimum notional requirements
func (filters *SymbolFilters) ValidateNotional(quantity, price float64) error {
	if filters.MinNotional == nil {
		return nil // No notional filter
	}

	notionalValue := quantity * price
	if notionalValue < filters.MinNotional.MinNotional {
		return fmt.Errorf("notional value %.2f is below minimum %.2f for %s", notionalValue, filters.MinNotional.MinNotional, filters.Symbol)
	}

	return nil
}

// RefreshAllSymbols updates all symbol information (should be called periodically)
func (s *ExchangeInfoService) RefreshAllSymbols() error {
	return s.refreshSymbolInfo("") // Empty string triggers full refresh
}

// GetCacheStatus returns cache information for monitoring
func (s *ExchangeInfoService) GetCacheStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cached_symbols_count": len(s.symbolFilters),
		"last_full_update":     s.lastFullUpdate,
		"cache_timeout":        s.cacheTimeout,
		"cache_age":            time.Since(s.lastFullUpdate),
	}
}
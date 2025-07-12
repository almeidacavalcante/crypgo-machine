package service

import (
	"crypgo-machine/src/domain/vo"
	"time"
)

// MarketDataSource abstracts the source of market data for trading operations
type MarketDataSource interface {
	// GetMarketData retrieves klines for the given symbol
	GetMarketData(symbol string) ([]vo.Kline, error)
	
	// GetCurrentTime returns the current time in the context of the data source
	// For live data, this is the current system time
	// For historical data, this is the timestamp of the current kline being processed
	GetCurrentTime() time.Time
}
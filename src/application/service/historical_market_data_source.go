package service

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

// HistoricalMarketDataSource implements MarketDataSource using pre-loaded historical data
type HistoricalMarketDataSource struct {
	historicalData []vo.Kline
	currentIndex   int
	windowSize     int // Number of klines to return (like the limit in live data)
}

// NewHistoricalMarketDataSource creates a new HistoricalMarketDataSource
func NewHistoricalMarketDataSource(historicalData []vo.Kline, windowSize int) *HistoricalMarketDataSource {
	return &HistoricalMarketDataSource{
		historicalData: historicalData,
		currentIndex:   0,
		windowSize:     windowSize,
	}
}

// GetMarketData returns a window of historical klines ending at the current index
// Note: intervalSeconds is ignored for historical data as the data is already filtered by interval
func (s *HistoricalMarketDataSource) GetMarketData(symbol string, intervalSeconds int) ([]vo.Kline, error) {
	if s.currentIndex >= len(s.historicalData) {
		return nil, fmt.Errorf("no more historical data available")
	}

	// Calculate the start index for the window
	startIndex := s.currentIndex - s.windowSize + 1
	if startIndex < 0 {
		startIndex = 0
	}

	// Return the window of klines from startIndex to currentIndex (inclusive)
	endIndex := s.currentIndex + 1
	if endIndex > len(s.historicalData) {
		endIndex = len(s.historicalData)
	}

	return s.historicalData[startIndex:endIndex], nil
}

// GetCurrentTime returns the timestamp of the current kline being processed
func (s *HistoricalMarketDataSource) GetCurrentTime() time.Time {
	if s.currentIndex < len(s.historicalData) {
		return time.Unix(s.historicalData[s.currentIndex].CloseTime()/1000, 0)
	}
	return time.Now()
}

// AdvanceToNext moves to the next kline in the historical data
func (s *HistoricalMarketDataSource) AdvanceToNext() bool {
	if s.currentIndex+1 < len(s.historicalData) {
		s.currentIndex++
		return true
	}
	return false
}

// HasMoreData returns true if there's more historical data to process
func (s *HistoricalMarketDataSource) HasMoreData() bool {
	return s.currentIndex < len(s.historicalData)-1
}

// GetCurrentIndex returns the current index in the historical data
func (s *HistoricalMarketDataSource) GetCurrentIndex() int {
	return s.currentIndex
}

// GetTotalDataPoints returns the total number of data points available
func (s *HistoricalMarketDataSource) GetTotalDataPoints() int {
	return len(s.historicalData)
}
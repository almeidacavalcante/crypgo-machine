package external

import (
	"context"
	"crypgo-machine/src/domain/vo"
	"fmt"
	"strconv"
	"time"
	"github.com/adshao/go-binance/v2"
)

// BinanceHistoricalDataService fetches historical data from Binance API
type BinanceHistoricalDataService struct {
	client BinanceClientInterface
}

func NewBinanceHistoricalDataService(client BinanceClientInterface) *BinanceHistoricalDataService {
	return &BinanceHistoricalDataService{
		client: client,
	}
}

// GetYesterdayKlines fetches 1-minute klines for the previous day (yesterday)
func (s *BinanceHistoricalDataService) GetYesterdayKlines(symbol string) ([]vo.Kline, error) {
	// Calculate yesterday's date range
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	
	// Set to start of yesterday (00:00:00)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	
	// Set to end of yesterday (23:59:59)
	endOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())
	
	fmt.Printf("🔍 Fetching klines for %s from %s to %s\n", symbol, startOfYesterday.Format("2006-01-02 15:04:05"), endOfYesterday.Format("2006-01-02 15:04:05"))
	
	return s.GetKlinesForPeriod(symbol, startOfYesterday, endOfYesterday, "1m")
}

// GetLastWeekKlines fetches 1-minute klines for the last 7 days
func (s *BinanceHistoricalDataService) GetLastWeekKlines(symbol string) ([]vo.Kline, error) {
	// Calculate last week's date range
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	
	// Set to start of week ago (00:00:00)
	startOfWeek := time.Date(weekAgo.Year(), weekAgo.Month(), weekAgo.Day(), 0, 0, 0, 0, weekAgo.Location())
	
	// Set to end of yesterday (don't include today's incomplete data)
	yesterday := now.AddDate(0, 0, -1)
	endOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location())
	
	fmt.Printf("🔍 Fetching klines for %s from %s to %s (7 days)\n", symbol, startOfWeek.Format("2006-01-02 15:04:05"), endOfYesterday.Format("2006-01-02 15:04:05"))
	
	return s.GetKlinesForWeekPeriod(symbol, startOfWeek, endOfYesterday, "1m")
}

// GetKlinesForPeriod fetches klines for a specific time period
func (s *BinanceHistoricalDataService) GetKlinesForPeriod(symbol string, startTime, endTime time.Time, interval string) ([]vo.Kline, error) {
	ctx := context.Background()
	
	// For 1-minute intervals over 24 hours, we need 1440 klines
	// Binance API has a limit of 1000 klines per request, so we need 2 requests
	var allKlines []vo.Kline
	
	// First batch: get 1000 klines
	fmt.Printf("📊 Requesting first 1000 klines for %s with interval %s\n", symbol, interval)
	binanceKlines1, err := s.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(1000).
		Do(ctx)
	
	if err != nil {
		fmt.Printf("❌ Binance API error (first batch): %v\n", err)
		return nil, fmt.Errorf("failed to fetch first batch of klines from Binance: %w", err)
	}
	fmt.Printf("✅ Received %d klines from Binance (first batch)\n", len(binanceKlines1))
	
	// Convert first batch
	for _, bk := range binanceKlines1 {
		kline, err := s.convertBinanceKlineToVOKline(bk)
		if err != nil {
			return nil, fmt.Errorf("failed to convert kline: %w", err)
		}
		allKlines = append(allKlines, kline)
	}
	
	// Second batch: get remaining klines (up to 440 more)
	if len(binanceKlines1) == 1000 {
		fmt.Printf("📊 Requesting remaining klines for %s with interval %s\n", symbol, interval)
		binanceKlines2, err := s.client.NewKlinesService().
			Symbol(symbol).
			Interval(interval).
			Limit(440).
			Do(ctx)
		
		if err != nil {
			fmt.Printf("❌ Binance API error (second batch): %v\n", err)
			return nil, fmt.Errorf("failed to fetch second batch of klines from Binance: %w", err)
		}
		fmt.Printf("✅ Received %d klines from Binance (second batch)\n", len(binanceKlines2))
		
		// Convert second batch
		for _, bk := range binanceKlines2 {
			kline, err := s.convertBinanceKlineToVOKline(bk)
			if err != nil {
				return nil, fmt.Errorf("failed to convert kline: %w", err)
			}
			allKlines = append(allKlines, kline)
		}
	}
	
	fmt.Printf("📋 Total converted klines: %d\n", len(allKlines))
	return allKlines, nil
}

// GetKlinesForWeekPeriod fetches klines for a week period (7 days = ~10080 klines)
func (s *BinanceHistoricalDataService) GetKlinesForWeekPeriod(symbol string, startTime, endTime time.Time, interval string) ([]vo.Kline, error) {
	ctx := context.Background()
	
	// For 1-minute intervals over 7 days, we need ~10080 klines
	// Binance API has a limit of 1000 klines per request, so we need multiple requests
	var allKlines []vo.Kline
	
	// We'll make multiple requests to get all the data
	requestCount := 0
	const maxRequests = 15 // Safety limit to avoid infinite loops
	
	for requestCount < maxRequests {
		requestCount++
		limit := 1000
		
		fmt.Printf("📊 Requesting klines batch %d (limit: %d) for %s with interval %s\n", requestCount, limit, symbol, interval)
		
		binanceKlines, err := s.client.NewKlinesService().
			Symbol(symbol).
			Interval(interval).
			Limit(limit).
			Do(ctx)
		
		if err != nil {
			fmt.Printf("❌ Binance API error (batch %d): %v\n", requestCount, err)
			return nil, fmt.Errorf("failed to fetch batch %d of klines from Binance: %w", requestCount, err)
		}
		fmt.Printf("✅ Received %d klines from Binance (batch %d)\n", len(binanceKlines), requestCount)
		
		if len(binanceKlines) == 0 {
			break // No more data
		}
		
		// Convert batch
		for _, bk := range binanceKlines {
			kline, err := s.convertBinanceKlineToVOKline(bk)
			if err != nil {
				return nil, fmt.Errorf("failed to convert kline: %w", err)
			}
			allKlines = append(allKlines, kline)
		}
		
		// If we got less than the limit, we've reached the end
		if len(binanceKlines) < limit {
			break
		}
		
		// If we have enough data for a week (~10080 klines), stop
		if len(allKlines) >= 10080 {
			fmt.Printf("📋 Reached target klines count: %d\n", len(allKlines))
			break
		}
	}
	
	fmt.Printf("📋 Total converted klines for week: %d\n", len(allKlines))
	return allKlines, nil
}

// convertBinanceKlineToVOKline converts a Binance Kline to domain vo.Kline
func (s *BinanceHistoricalDataService) convertBinanceKlineToVOKline(bk *binance.Kline) (vo.Kline, error) {
	open, err := strconv.ParseFloat(bk.Open, 64)
	if err != nil {
		return vo.Kline{}, fmt.Errorf("invalid open price: %w", err)
	}
	
	close, err := strconv.ParseFloat(bk.Close, 64)
	if err != nil {
		return vo.Kline{}, fmt.Errorf("invalid close price: %w", err)
	}
	
	high, err := strconv.ParseFloat(bk.High, 64)
	if err != nil {
		return vo.Kline{}, fmt.Errorf("invalid high price: %w", err)
	}
	
	low, err := strconv.ParseFloat(bk.Low, 64)
	if err != nil {
		return vo.Kline{}, fmt.Errorf("invalid low price: %w", err)
	}
	
	volume, err := strconv.ParseFloat(bk.Volume, 64)
	if err != nil {
		return vo.Kline{}, fmt.Errorf("invalid volume: %w", err)
	}
	
	return vo.NewKline(open, close, high, low, volume, bk.CloseTime)
}
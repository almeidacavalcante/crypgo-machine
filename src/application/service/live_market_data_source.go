package service

import (
	"context"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
	"strconv"
	"time"
)

// LiveMarketDataSource implements MarketDataSource using real-time Binance API data
type LiveMarketDataSource struct {
	client external.BinanceClientInterface
}

// NewLiveMarketDataSource creates a new LiveMarketDataSource
func NewLiveMarketDataSource(client external.BinanceClientInterface) *LiveMarketDataSource {
	return &LiveMarketDataSource{
		client: client,
	}
}

// GetMarketData fetches the latest klines from Binance API with dynamic interval
func (s *LiveMarketDataSource) GetMarketData(symbol string, intervalSeconds int) ([]vo.Kline, error) {
	// Convert seconds to Binance interval format
	interval, err := external.SecondsToInterval(intervalSeconds)
	if err != nil {
		return nil, fmt.Errorf("invalid interval %d seconds: %v", intervalSeconds, err)
	}
	
	binanceKlines, err := s.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(100).
		Do(context.Background())

	if err != nil {
		return nil, err
	}

	klines := make([]vo.Kline, len(binanceKlines))
	for i, bkline := range binanceKlines {
		openPrice, _ := strconv.ParseFloat(bkline.Open, 64)
		closePrice, _ := strconv.ParseFloat(bkline.Close, 64)
		highPrice, _ := strconv.ParseFloat(bkline.High, 64)
		lowPrice, _ := strconv.ParseFloat(bkline.Low, 64)
		volumePrice, _ := strconv.ParseFloat(bkline.Volume, 64)

		kline, err := vo.NewKline(openPrice, closePrice, highPrice, lowPrice, volumePrice, bkline.CloseTime)
		if err != nil {
			return nil, err
		}
		klines[i] = kline
	}

	return klines, nil
}

// GetCurrentTime returns the current system time for live trading
func (s *LiveMarketDataSource) GetCurrentTime() time.Time {
	return time.Now()
}

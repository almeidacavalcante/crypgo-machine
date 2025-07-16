package service

import (
	"crypgo-machine/src/infra/external"
	"testing"
)

func TestLiveMarketDataSource_GetMarketData_WithDifferentIntervals(t *testing.T) {
	// Use fake client for testing
	client := external.NewBinanceClientFake()
	dataSource := NewLiveMarketDataSource(client)
	
	testCases := []struct {
		name            string
		intervalSeconds int
		expectedError   bool
		description     string
	}{
		{"1 minute", 60, false, "Should work with 1m interval"},
		{"5 minutes", 300, false, "Should work with 5m interval"},
		{"15 minutes", 900, false, "Should work with 15m interval"},
		{"1 hour", 3600, false, "Should work with 1h interval"},
		{"Invalid interval", 120, true, "Should fail with unsupported interval"},
		{"Another invalid", 999, true, "Should fail with unsupported interval"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			klines, err := dataSource.GetMarketData("BTCBRL", tc.intervalSeconds)
			
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error for interval %d seconds, but got none", tc.intervalSeconds)
				}
				if klines != nil {
					t.Errorf("Expected nil klines on error, but got %d klines", len(klines))
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for interval %d seconds: %v", tc.intervalSeconds, err)
				}
				if klines == nil {
					t.Errorf("Expected klines for interval %d seconds, but got nil", tc.intervalSeconds)
				}
			}
		})
	}
}

func TestLiveMarketDataSource_GetCurrentTime(t *testing.T) {
	client := external.NewBinanceClientFake()
	dataSource := NewLiveMarketDataSource(client)
	
	currentTime := dataSource.GetCurrentTime()
	if currentTime.IsZero() {
		t.Error("GetCurrentTime should return a non-zero time")
	}
}
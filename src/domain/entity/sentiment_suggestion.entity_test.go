package entity

import (
	"testing"

	"crypgo-machine/src/domain/vo"
)

// Kline intervals Binance offers, in seconds. A trading bot can only run on
// one of these (see external.SecondsToInterval), so a suggested interval
// outside this list cannot be applied to a bot.
var binanceIntervalSeconds = map[int]bool{
	60: true, 180: true, 300: true, 900: true, 1800: true,
	3600: true, 7200: true, 14400: true, 21600: true, 28800: true, 43200: true,
	86400: true, 259200: true, 604800: true, 2592000: true,
}

func TestGenerateSuggestions_IntervalIsSupportedByBinance(t *testing.T) {
	levels := []vo.SentimentLevel{
		vo.VeryBullish, vo.Bullish, vo.Neutral, vo.Bearish, vo.VeryBearish,
		vo.SentimentLevel("unknown"),
	}

	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			multiplier, threshold, interval := generateSuggestions(level)

			if !binanceIntervalSeconds[interval] {
				t.Errorf("suggested interval %d seconds is not a Binance kline interval", interval)
			}
			if multiplier <= 0 {
				t.Errorf("expected a positive multiplier, got %v", multiplier)
			}
			if threshold <= 0 {
				t.Errorf("expected a positive profit threshold, got %v", threshold)
			}
		})
	}
}

// More bearish sentiment must never suggest checking the market more often.
func TestGenerateSuggestions_IntervalGrowsAsSentimentFalls(t *testing.T) {
	ordered := []vo.SentimentLevel{vo.VeryBullish, vo.Bullish, vo.Neutral, vo.Bearish, vo.VeryBearish}

	previous := 0
	for _, level := range ordered {
		_, _, interval := generateSuggestions(level)
		if interval < previous {
			t.Errorf("level %s suggests %d seconds, shorter than the more bullish level before it (%d)", level, interval, previous)
		}
		previous = interval
	}
}

package vo

import (
	"fmt"
	"strings"
)

type SentimentLevel string

const (
	VeryBearish SentimentLevel = "very_bearish"
	Bearish     SentimentLevel = "bearish"
	Neutral     SentimentLevel = "neutral"
	Bullish     SentimentLevel = "bullish"
	VeryBullish SentimentLevel = "very_bullish"
)

func NewSentimentLevel(level string) (SentimentLevel, error) {
	normalized := strings.ToLower(strings.TrimSpace(level))
	
	switch normalized {
	case string(VeryBearish):
		return VeryBearish, nil
	case string(Bearish):
		return Bearish, nil
	case string(Neutral):
		return Neutral, nil
	case string(Bullish):
		return Bullish, nil
	case string(VeryBullish):
		return VeryBullish, nil
	default:
		return "", fmt.Errorf("invalid sentiment level: %s", level)
	}
}

func (s SentimentLevel) String() string {
	return string(s)
}

func (s SentimentLevel) GetValue() string {
	return string(s)
}

func (s SentimentLevel) GetDisplayName() string {
	switch s {
	case VeryBearish:
		return "Very Bearish"
	case Bearish:
		return "Bearish"
	case Neutral:
		return "Neutral"
	case Bullish:
		return "Bullish"
	case VeryBullish:
		return "Very Bullish"
	default:
		return "Unknown"
	}
}

func (s SentimentLevel) GetNumericValue() float64 {
	switch s {
	case VeryBearish:
		return -1.0
	case Bearish:
		return -0.5
	case Neutral:
		return 0.0
	case Bullish:
		return 0.5
	case VeryBullish:
		return 1.0
	default:
		return 0.0
	}
}

func SentimentLevelFromScore(score float64) SentimentLevel {
	if score >= 0.3 {
		return VeryBullish
	} else if score >= 0.1 {
		return Bullish
	} else if score <= -0.3 {
		return VeryBearish
	} else if score <= -0.1 {
		return Bearish
	}
	return Neutral
}
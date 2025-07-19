package vo

import (
	"fmt"
	"math"
)

type SentimentScore struct {
	value float64
}

func NewSentimentScore(score float64) (*SentimentScore, error) {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return nil, fmt.Errorf("sentiment score cannot be NaN or Inf")
	}
	
	// Normalize score to -1.0 to 1.0 range based on magnitude
	normalizedScore := normalizeScore(score)
	
	return &SentimentScore{value: normalizedScore}, nil
}

// NewSentimentScoreNormalized creates a score that's already in the -1.0 to 1.0 range
func NewSentimentScoreNormalized(score float64) (*SentimentScore, error) {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return nil, fmt.Errorf("sentiment score cannot be NaN or Inf")
	}
	
	if score < -1.0 || score > 1.0 {
		return nil, fmt.Errorf("normalized score must be between -1.0 and 1.0, got: %.3f", score)
	}
	
	return &SentimentScore{value: score}, nil
}

// normalizeScore converts any range of scores to -1.0 to 1.0
// Uses a consistent sigmoid approach that preserves magnitude ordering
func normalizeScore(score float64) float64 {
	// Always apply normalization to preserve monotonic ordering
	// Don't treat values in [-1,1] as special case since that breaks ordering
	
	// Use scaled tanh that maps common sentiment ranges nicely:
	// 0 → 0, 1 → 0.46, 2 → 0.76, 4 → 0.96, 8 → 0.999, 80 → ~1.0
	// Scale factor controls the sensitivity
	scaleFactor := 2.0
	
	return math.Tanh(score / scaleFactor)
}

func (s *SentimentScore) GetValue() float64 {
	return s.value
}

func (s *SentimentScore) IsPositive() bool {
	return s.value > 0
}

func (s *SentimentScore) IsNegative() bool {
	return s.value < 0
}

func (s *SentimentScore) IsNeutral() bool {
	return s.value == 0
}

func (s *SentimentScore) GetLevel() SentimentLevel {
	return SentimentLevelFromScore(s.value)
}

func (s *SentimentScore) GetConfidence() float64 {
	return math.Abs(s.value)
}

func (s *SentimentScore) String() string {
	if s.value >= 0 {
		return fmt.Sprintf("+%.3f", s.value)
	}
	return fmt.Sprintf("%.3f", s.value)
}
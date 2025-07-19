package vo

import (
	"math"
	"testing"
)

func TestSentimentScore_Normalization(t *testing.T) {
	tests := []struct {
		name          string
		inputScore    float64
		expectInRange bool
		expectSign    string
	}{
		{"Small positive", 0.5, true, "positive"},
		{"Small negative", -0.3, true, "negative"},
		{"Medium positive", 2.0, true, "positive"},
		{"Medium negative", -3.0, true, "negative"},
		{"Large positive", 8.0, true, "positive"},
		{"Huge positive", 80.0, true, "positive"},
		{"Large negative", -15.0, true, "negative"},
		{"Zero", 0.0, true, "neutral"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewSentimentScore(tt.inputScore)
			if err != nil {
				t.Fatalf("NewSentimentScore failed: %v", err)
			}
			
			value := score.GetValue()
			
			// Check if in valid range
			if tt.expectInRange {
				if value < -1.0 || value > 1.0 {
					t.Errorf("Score %f normalized to %f, which is outside [-1.0, 1.0] range", 
						tt.inputScore, value)
				}
			}
			
			// Check sign preservation
			switch tt.expectSign {
			case "positive":
				if value <= 0 {
					t.Errorf("Expected positive score, got %f", value)
				}
			case "negative":
				if value >= 0 {
					t.Errorf("Expected negative score, got %f", value)
				}
			case "neutral":
				if value != 0 {
					t.Errorf("Expected zero score, got %f", value)
				}
			}
		})
	}
}

func TestSentimentScore_MagnitudePreservation(t *testing.T) {
	// Test that larger absolute values result in larger normalized values
	scores := []float64{1.0, 2.0, 5.0, 8.0, 20.0, 80.0}
	
	var normalizedScores []float64
	for _, rawScore := range scores {
		score, err := NewSentimentScore(rawScore)
		if err != nil {
			t.Fatalf("NewSentimentScore failed for %f: %v", rawScore, err)
		}
		normalizedScores = append(normalizedScores, score.GetValue())
	}
	
	// Check that normalized scores are monotonically increasing
	for i := 1; i < len(normalizedScores); i++ {
		if normalizedScores[i] <= normalizedScores[i-1] {
			t.Errorf("Normalization lost magnitude ordering: scores[%d]=%.6f should be > scores[%d]=%.6f (raw: %f vs %f)",
				i, normalizedScores[i], i-1, normalizedScores[i-1], scores[i], scores[i-1])
		}
	}
	
	// Print the mapping for verification
	t.Logf("Score normalization mapping:")
	for i, rawScore := range scores {
		t.Logf("  %6.1f → %8.6f", rawScore, normalizedScores[i])
	}
}

func TestSentimentScore_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		inputScore  float64
		expectError bool
	}{
		{"NaN", math.NaN(), true},
		{"Positive Infinity", math.Inf(1), true},
		{"Negative Infinity", math.Inf(-1), true},
		{"Very large positive", 1e6, false},
		{"Very large negative", -1e6, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewSentimentScore(tt.inputScore)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input %f, but got none", tt.inputScore)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %f: %v", tt.inputScore, err)
				}
				if score != nil {
					value := score.GetValue()
					if value < -1.0 || value > 1.0 {
						t.Errorf("Score %f normalized to %f, outside valid range", tt.inputScore, value)
					}
				}
			}
		})
	}
}

func TestSentimentScoreNormalized_Validation(t *testing.T) {
	tests := []struct {
		name        string
		inputScore  float64
		expectError bool
	}{
		{"Valid positive", 0.8, false},
		{"Valid negative", -0.3, false},
		{"Valid zero", 0.0, false},
		{"Valid max", 1.0, false},
		{"Valid min", -1.0, false},
		{"Invalid too high", 1.1, true},
		{"Invalid too low", -1.5, true},
		{"Invalid way too high", 10.0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewSentimentScoreNormalized(tt.inputScore)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for normalized input %f, but got none", tt.inputScore)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for normalized input %f: %v", tt.inputScore, err)
				}
				if score != nil && score.GetValue() != tt.inputScore {
					t.Errorf("Expected value %f, got %f", tt.inputScore, score.GetValue())
				}
			}
		})
	}
}

// Benchmark the normalization function
func BenchmarkNormalizeScore(b *testing.B) {
	testScores := []float64{0.5, 2.0, 8.0, 80.0, -15.0, 1000.0}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, score := range testScores {
			normalizeScore(score)
		}
	}
}
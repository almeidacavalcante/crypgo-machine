package external

import "testing"

func TestSecondsToInterval(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
		hasError bool
	}{
		{"1 minute", 60, "1m", false},
		{"3 minutes", 180, "3m", false},
		{"5 minutes", 300, "5m", false},
		{"15 minutes", 900, "15m", false},
		{"30 minutes", 1800, "30m", false},
		{"1 hour", 3600, "1h", false},
		{"2 hours", 7200, "2h", false},
		{"4 hours", 14400, "4h", false},
		{"6 hours", 21600, "6h", false},
		{"8 hours", 28800, "8h", false},
		{"12 hours", 43200, "12h", false},
		{"1 day", 86400, "1d", false},
		{"3 days", 259200, "3d", false},
		{"1 week", 604800, "1w", false},
		{"1 month", 2592000, "1M", false},
		{"Invalid interval", 120, "", true},
		{"Another invalid", 999, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SecondsToInterval(tt.seconds)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for %d seconds, but got none", tt.seconds)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %d seconds: %v", tt.seconds, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s for %d seconds, got %s", tt.expected, tt.seconds, result)
				}
			}
		})
	}
}

func TestIntervalToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		expected int
		hasError bool
	}{
		{"1m", "1m", 60, false},
		{"3m", "3m", 180, false},
		{"5m", "5m", 300, false},
		{"15m", "15m", 900, false},
		{"30m", "30m", 1800, false},
		{"1h", "1h", 3600, false},
		{"2h", "2h", 7200, false},
		{"4h", "4h", 14400, false},
		{"6h", "6h", 21600, false},
		{"8h", "8h", 28800, false},
		{"12h", "12h", 43200, false},
		{"1d", "1d", 86400, false},
		{"3d", "3d", 259200, false},
		{"1w", "1w", 604800, false},
		{"1M", "1M", 2592000, false},
		{"Invalid interval", "2m", 0, true},
		{"Another invalid", "99h", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IntervalToSeconds(tt.interval)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for %s interval, but got none", tt.interval)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s interval: %v", tt.interval, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d for %s interval, got %d", tt.expected, tt.interval, result)
				}
			}
		})
	}
}

func TestIsValidInterval(t *testing.T) {
	validIntervals := []int{60, 180, 300, 900, 1800, 3600, 7200, 14400, 21600, 28800, 43200, 86400, 259200, 604800, 2592000}
	invalidIntervals := []int{30, 120, 240, 600, 1200, 7199, 999999}

	for _, interval := range validIntervals {
		if !IsValidInterval(interval) {
			t.Errorf("Expected %d seconds to be valid, but it was not", interval)
		}
	}

	for _, interval := range invalidIntervals {
		if IsValidInterval(interval) {
			t.Errorf("Expected %d seconds to be invalid, but it was valid", interval)
		}
	}
}

func TestGetSupportedIntervals(t *testing.T) {
	intervals := GetSupportedIntervals()
	expectedCount := 15 // Number of supported intervals
	
	if len(intervals) != expectedCount {
		t.Errorf("Expected %d supported intervals, got %d", expectedCount, len(intervals))
	}
	
	// Test that all returned intervals are valid
	for _, interval := range intervals {
		if !IsValidInterval(interval) {
			t.Errorf("GetSupportedIntervals returned invalid interval: %d", interval)
		}
	}
}
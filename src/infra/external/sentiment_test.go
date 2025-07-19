package external

import (
	"testing"
	"time"
)

func TestFearGreedClient_GetLatestIndex(t *testing.T) {
	client := NewFearGreedClient()
	
	data, err := client.GetLatestIndex()
	if err != nil {
		t.Logf("Fear & Greed API test failed (expected in test environment): %v", err)
		// Don't fail the test, just log the error as this requires internet access
		return
	}
	
	if data == nil {
		t.Error("Expected data, got nil")
		return
	}
	
	if data.Value < 0 || data.Value > 100 {
		t.Errorf("Expected value between 0-100, got %d", data.Value)
	}
	
	if data.Classification == "" {
		t.Error("Expected non-empty classification")
	}
	
	if data.Timestamp.IsZero() {
		t.Error("Expected valid timestamp")
	}
	
	// Test normalized score calculation
	normalizedScore := data.GetNormalizedScore()
	if normalizedScore < -1.0 || normalizedScore > 1.0 {
		t.Errorf("Expected normalized score between -1.0 and 1.0, got %f", normalizedScore)
	}
	
	t.Logf("Fear & Greed Index: %d (%s), Normalized: %.3f", 
		data.Value, data.Classification, normalizedScore)
}

func TestSentimentAnalyzer_AnalyzeText(t *testing.T) {
	analyzer := NewSentimentAnalyzer()
	
	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Positive sentiment",
			text:     "Bitcoin is showing bullish momentum with strong adoption and institutional support. Rally continues as prices surge.",
			expected: "positive",
		},
		{
			name:     "Negative sentiment", 
			text:     "Market crash and bearish sentiment as regulation concerns cause massive dump. Fear dominates trading.",
			expected: "negative",
		},
		{
			name:     "Neutral sentiment",
			text:     "Market analysis shows mixed signals. Technical charts indicate possible movement in either direction.",
			expected: "neutral",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.AnalyzeText(tc.text)
			
			if result.Classification != tc.expected {
				t.Errorf("Expected classification %s, got %s", tc.expected, result.Classification)
			}
			
			if result.Score < -1.0 || result.Score > 1.0 {
				t.Errorf("Expected score between -1.0 and 1.0, got %f", result.Score)
			}
			
			t.Logf("Text: %s\nScore: %.3f, Classification: %s, Confidence: %.3f", 
				tc.text[:50]+"...", result.Score, result.Classification, result.Confidence)
		})
	}
}

func TestRSSFeedReader_FetchFeed(t *testing.T) {
	reader := NewRSSFeedReader()
	
	// Test with a reliable RSS feed (this might fail in test environment)
	newsItems, err := reader.FetchFeed("https://feeds.feedburner.com/oreilly/radar", "TestFeed")
	if err != nil {
		t.Logf("RSS feed test failed (expected in test environment): %v", err)
		// Don't fail the test, just log the error as this requires internet access
		return
	}
	
	if len(newsItems) == 0 {
		t.Error("Expected at least one news item")
		return
	}
	
	firstItem := newsItems[0]
	if firstItem.Title == "" {
		t.Error("Expected non-empty title")
	}
	
	if firstItem.Source != "TestFeed" {
		t.Errorf("Expected source 'TestFeed', got '%s'", firstItem.Source)
	}
	
	if firstItem.Content == "" {
		t.Error("Expected non-empty content")
	}
	
	t.Logf("Retrieved %d news items from test feed", len(newsItems))
}

func TestSentimentAggregator_QuickAnalysis(t *testing.T) {
	aggregator := NewSentimentAggregator()
	
	result, err := aggregator.QuickAnalysis()
	if err != nil {
		t.Logf("Quick analysis test failed (expected in test environment): %v", err)
		// Don't fail the test, just log the error as this requires internet access
		return
	}
	
	if result == nil {
		t.Error("Expected result, got nil")
		return
	}
	
	if result.OverallScore < -1.0 || result.OverallScore > 1.0 {
		t.Errorf("Expected overall score between -1.0 and 1.0, got %f", result.OverallScore)
	}
	
	expectedLevels := []string{"very_bullish", "bullish", "neutral", "bearish", "very_bearish"}
	levelFound := false
	for _, level := range expectedLevels {
		if result.SentimentLevel == level {
			levelFound = true
			break
		}
	}
	
	if !levelFound {
		t.Errorf("Expected sentiment level to be one of %v, got %s", expectedLevels, result.SentimentLevel)
	}
	
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		t.Errorf("Expected confidence between 0.0 and 1.0, got %f", result.Confidence)
	}
	
	if result.Timestamp.IsZero() {
		t.Error("Expected valid timestamp")
	}
	
	t.Logf("Quick Analysis Result - Sentiment: %s, Score: %.3f, Confidence: %.3f", 
		result.SentimentLevel, result.OverallScore, result.Confidence)
}

func TestNewsItem_Creation(t *testing.T) {
	newsItem := NewsItem{
		Title:       "Bitcoin Shows Strong Performance",
		Description: "Analysis of recent market trends",
		Link:        "https://example.com/news/1",
		Source:      "TestSource",
		PublishedAt: time.Now(),
		Content:     "Bitcoin Shows Strong Performance Analysis of recent market trends",
	}
	
	if newsItem.Title == "" {
		t.Error("Expected non-empty title")
	}
	
	if newsItem.Source == "" {
		t.Error("Expected non-empty source")
	}
	
	if newsItem.Content == "" {
		t.Error("Expected non-empty content")
	}
	
	analyzer := NewSentimentAnalyzer()
	result := analyzer.AnalyzeText(newsItem.Content)
	
	t.Logf("News item sentiment analysis - Score: %.3f, Classification: %s", 
		result.Score, result.Classification)
}

func TestSentimentLevels(t *testing.T) {
	testCases := []struct {
		score          float64
		expectedLevel  string
	}{
		{0.5, "very_bullish"},
		{0.2, "bullish"},
		{0.0, "neutral"},
		{-0.2, "bearish"},
		{-0.5, "very_bearish"},
	}
	
	aggregator := NewSentimentAggregator()
	
	for _, tc := range testCases {
		level := aggregator.determineSentimentLevel(tc.score)
		if level != tc.expectedLevel {
			t.Errorf("Score %.1f: expected level %s, got %s", tc.score, tc.expectedLevel, level)
		}
	}
}

// Benchmark tests for performance
func BenchmarkSentimentAnalyzer_AnalyzeText(b *testing.B) {
	analyzer := NewSentimentAnalyzer()
	text := "Bitcoin showing bullish momentum with strong institutional adoption and positive market sentiment driving significant rally as investors show optimistic outlook"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AnalyzeText(text)
	}
}
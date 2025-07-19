package vo

import "fmt"

type SentimentSources struct {
	fearGreedIndex   int
	newsScore        float64
	redditScore      float64
	socialScore      float64
}

func NewSentimentSources(fearGreed int, news, reddit, social float64) (*SentimentSources, error) {
	if fearGreed < 0 || fearGreed > 100 {
		return nil, fmt.Errorf("fear & greed index must be between 0 and 100, got: %d", fearGreed)
	}
	
	if news < -1.0 || news > 1.0 {
		return nil, fmt.Errorf("news score must be between -1.0 and 1.0, got: %.3f", news)
	}
	
	if reddit < -1.0 || reddit > 1.0 {
		return nil, fmt.Errorf("reddit score must be between -1.0 and 1.0, got: %.3f", reddit)
	}
	
	if social < -1.0 || social > 1.0 {
		return nil, fmt.Errorf("social score must be between -1.0 and 1.0, got: %.3f", social)
	}
	
	return &SentimentSources{
		fearGreedIndex: fearGreed,
		newsScore:      news,
		redditScore:    reddit,
		socialScore:    social,
	}, nil
}

func (s *SentimentSources) GetFearGreedIndex() int {
	return s.fearGreedIndex
}

func (s *SentimentSources) GetNewsScore() float64 {
	return s.newsScore
}

func (s *SentimentSources) GetRedditScore() float64 {
	return s.redditScore
}

func (s *SentimentSources) GetSocialScore() float64 {
	return s.socialScore
}

func (s *SentimentSources) GetFearGreedNormalized() float64 {
	// Convert 0-100 scale to -1.0 to 1.0 scale
	return (float64(s.fearGreedIndex) - 50.0) / 50.0
}

func (s *SentimentSources) CalculateOverallScore() (*SentimentScore, error) {
	// Weighted average calculation
	weights := struct {
		fearGreed float64
		news      float64
		reddit    float64
		social    float64
	}{
		fearGreed: 0.4,
		news:      0.35,
		reddit:    0.15,
		social:    0.1,
	}
	
	fgNormalized := s.GetFearGreedNormalized()
	
	overall := (fgNormalized * weights.fearGreed) +
		(s.newsScore * weights.news) +
		(s.redditScore * weights.reddit) +
		(s.socialScore * weights.social)
	
	return NewSentimentScore(overall)
}

func (s *SentimentSources) GetFearGreedClassification() string {
	switch {
	case s.fearGreedIndex >= 75:
		return "Extreme Greed"
	case s.fearGreedIndex >= 55:
		return "Greed"
	case s.fearGreedIndex >= 45:
		return "Neutral"
	case s.fearGreedIndex >= 25:
		return "Fear"
	default:
		return "Extreme Fear"
	}
}
package external

import (
	"fmt"
	"time"
)

type SentimentAggregator struct {
	fearGreedClient *FearGreedClient
	rssReader       *RSSFeedReader
	analyzer        *SentimentAnalyzer
}

type AggregatedSentiment struct {
	OverallScore    float64                `json:"overall_score"`
	SentimentLevel  string                 `json:"sentiment_level"`
	Confidence      float64                `json:"confidence"`
	Timestamp       time.Time              `json:"timestamp"`
	Sources         SentimentSources       `json:"sources"`
	Reasoning       string                 `json:"reasoning"`
	Recommendation  string                 `json:"recommendation"`
}

type SentimentSources struct {
	FearGreedIndex *FearGreedData         `json:"fear_greed_index"`
	NewsAnalysis   *NewsAnalysisResult    `json:"news_analysis"`
	RedditScore    float64                `json:"reddit_score"`
	WeightedScores WeightedScores         `json:"weighted_scores"`
}

type WeightedScores struct {
	FearGreedWeight float64 `json:"fear_greed_weight"`
	NewsWeight      float64 `json:"news_weight"`
	RedditWeight    float64 `json:"reddit_weight"`
	FearGreedScore  float64 `json:"fear_greed_score"`
	NewsScore       float64 `json:"news_score"`
}

// Weights as defined in the plan
const (
	FearGreedWeight = 0.4  // 40%
	NewsWeight      = 0.35 // 35%
	RedditWeight    = 0.25 // 25%
)

func NewSentimentAggregator() *SentimentAggregator {
	return &SentimentAggregator{
		fearGreedClient: NewFearGreedClient(),
		rssReader:       NewRSSFeedReader(),
		analyzer:        NewSentimentAnalyzer(),
	}
}

func (s *SentimentAggregator) CollectAndAnalyze() (*AggregatedSentiment, error) {
	timestamp := time.Now()
	
	// 1. Fetch Fear & Greed Index
	fearGreedData, err := s.fearGreedClient.GetLatestIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Fear & Greed Index: %w", err)
	}
	
	// 2. Fetch and analyze news from RSS feeds
	recentNews, err := s.rssReader.FetchRecentNews(24) // Last 24 hours
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS news: %w", err)
	}
	
	newsAnalysis := s.analyzer.AnalyzeNews(recentNews)
	
	// 3. Calculate Reddit-specific score from news analysis
	redditScore := s.extractRedditScore(&newsAnalysis)
	
	// 4. Aggregate scores using weighted average
	aggregated := s.calculateAggregatedScore(fearGreedData, &newsAnalysis, redditScore)
	aggregated.Timestamp = timestamp
	
	// 5. Generate reasoning and recommendation
	aggregated.Reasoning = s.generateReasoning(fearGreedData, &newsAnalysis, redditScore)
	aggregated.Recommendation = s.generateRecommendation(aggregated.OverallScore)
	
	return aggregated, nil
}

func (s *SentimentAggregator) calculateAggregatedScore(
	fearGreed *FearGreedData,
	newsAnalysis *NewsAnalysisResult,
	redditScore float64,
) *AggregatedSentiment {
	
	// Convert Fear & Greed Index to normalized score (-1 to +1)
	fearGreedScore := fearGreed.GetNormalizedScore()
	
	// News score is already normalized (-1 to +1)
	newsScore := newsAnalysis.OverallScore
	
	// Calculate weighted average as defined in the plan
	overallScore := (fearGreedScore * FearGreedWeight) +
		(newsScore * NewsWeight) +
		(redditScore * RedditWeight)
	
	// Determine sentiment level
	sentimentLevel := s.determineSentimentLevel(overallScore)
	
	// Calculate confidence based on data quality
	confidence := s.calculateConfidence(fearGreed, newsAnalysis, redditScore)
	
	return &AggregatedSentiment{
		OverallScore:   overallScore,
		SentimentLevel: sentimentLevel,
		Confidence:     confidence,
		Sources: SentimentSources{
			FearGreedIndex: fearGreed,
			NewsAnalysis:   newsAnalysis,
			RedditScore:    redditScore,
			WeightedScores: WeightedScores{
				FearGreedWeight: FearGreedWeight,
				NewsWeight:      NewsWeight,
				RedditWeight:    RedditWeight,
				FearGreedScore:  fearGreedScore,
				NewsScore:       newsScore,
			},
		},
	}
}

func (s *SentimentAggregator) extractRedditScore(newsAnalysis *NewsAnalysisResult) float64 {
	redditSources := []string{"RedditCrypto", "RedditCryptoTop"}
	var totalScore float64
	var count int
	
	for _, source := range redditSources {
		if sourceData, exists := newsAnalysis.SourceBreakdown[source]; exists {
			totalScore += sourceData.Score
			count++
		}
	}
	
	if count > 0 {
		return totalScore / float64(count)
	}
	
	return 0.0 // Neutral if no Reddit data
}

func (s *SentimentAggregator) determineSentimentLevel(score float64) string {
	// Levels as defined in the plan
	switch {
	case score > 0.3:
		return "very_bullish"
	case score > 0.1:
		return "bullish"
	case score < -0.3:
		return "very_bearish"
	case score < -0.1:
		return "bearish"
	default:
		return "neutral"
	}
}

func (s *SentimentAggregator) calculateConfidence(
	fearGreed *FearGreedData,
	newsAnalysis *NewsAnalysisResult,
	redditScore float64,
) float64 {
	
	var confidenceFactors []float64
	
	// Fear & Greed confidence (higher for extreme values)
	fearGreedConf := 0.7 // Base confidence
	if fearGreed.IsExtremeFear() || fearGreed.IsExtremeGreed() {
		fearGreedConf = 0.9
	}
	confidenceFactors = append(confidenceFactors, fearGreedConf)
	
	// News confidence based on article count
	newsConf := 0.5
	if newsAnalysis.TotalArticles >= 10 {
		newsConf = 0.8
	} else if newsAnalysis.TotalArticles >= 5 {
		newsConf = 0.7
	}
	confidenceFactors = append(confidenceFactors, newsConf)
	
	// Reddit confidence
	redditConf := 0.6
	if redditScore != 0 {
		redditConf = 0.7
	}
	confidenceFactors = append(confidenceFactors, redditConf)
	
	// Calculate weighted average confidence
	var totalConf float64
	for _, conf := range confidenceFactors {
		totalConf += conf
	}
	
	return totalConf / float64(len(confidenceFactors))
}

func (s *SentimentAggregator) generateReasoning(
	fearGreed *FearGreedData,
	newsAnalysis *NewsAnalysisResult,
	redditScore float64,
) string {
	
	reasoning := fmt.Sprintf(
		"Market sentiment analysis based on Fear & Greed Index (%d - %s), ",
		fearGreed.Value, fearGreed.Classification,
	)
	
	reasoning += fmt.Sprintf(
		"%d news articles analyzed (%d positive, %d negative), ",
		newsAnalysis.TotalArticles,
		newsAnalysis.PositiveArticles,
		newsAnalysis.NegativeArticles,
	)
	
	if redditScore > 0.1 {
		reasoning += "positive Reddit sentiment. "
	} else if redditScore < -0.1 {
		reasoning += "negative Reddit sentiment. "
	} else {
		reasoning += "neutral Reddit sentiment. "
	}
	
	// Add specific insights
	if fearGreed.IsExtremeFear() {
		reasoning += "Extreme fear levels suggest potential buying opportunity. "
	} else if fearGreed.IsExtremeGreed() {
		reasoning += "Extreme greed levels suggest caution and potential profit-taking. "
	}
	
	return reasoning
}

func (s *SentimentAggregator) generateRecommendation(overallScore float64) string {
	switch {
	case overallScore > 0.3:
		return "increase_exposure"
	case overallScore > 0.1:
		return "normal_plus"
	case overallScore < -0.3:
		return "minimal_exposure"
	case overallScore < -0.1:
		return "reduce_exposure"
	default:
		return "maintain"
	}
}

// QuickAnalysis performs a lightweight sentiment check without full data collection
func (s *SentimentAggregator) QuickAnalysis() (*AggregatedSentiment, error) {
	fearGreedData, err := s.fearGreedClient.GetLatestIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Fear & Greed Index for quick analysis: %w", err)
	}
	
	// Create simplified analysis based only on Fear & Greed
	fearGreedScore := fearGreedData.GetNormalizedScore()
	
	return &AggregatedSentiment{
		OverallScore:   fearGreedScore * 0.7, // Reduce impact since it's only one source
		SentimentLevel: s.determineSentimentLevel(fearGreedScore * 0.7),
		Confidence:     0.5, // Lower confidence for quick analysis
		Timestamp:      time.Now(),
		Sources: SentimentSources{
			FearGreedIndex: fearGreedData,
			WeightedScores: WeightedScores{
				FearGreedWeight: 1.0,
				FearGreedScore:  fearGreedScore,
			},
		},
		Reasoning:      fmt.Sprintf("Quick analysis based only on Fear & Greed Index: %d (%s)", fearGreedData.Value, fearGreedData.Classification),
		Recommendation: s.generateRecommendation(fearGreedScore * 0.7),
	}, nil
}
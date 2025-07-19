package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
	"time"
)

type MarketSentimentService struct {
	aggregator *external.SentimentAggregator
}

type SentimentCollectionResult struct {
	Suggestion *entity.SentimentSuggestion
	Sources    *vo.SentimentSources
	Reasoning  string
	Confidence float64
}

func NewMarketSentimentService() *MarketSentimentService {
	return &MarketSentimentService{
		aggregator: external.NewSentimentAggregator(),
	}
}

// CollectMarketSentiment performs full sentiment analysis and creates domain entities
func (s *MarketSentimentService) CollectMarketSentiment() (*SentimentCollectionResult, error) {
	// Collect and analyze sentiment data
	aggregated, err := s.aggregator.CollectAndAnalyze()
	if err != nil {
		return nil, fmt.Errorf("failed to collect sentiment data: %w", err)
	}
	
	// Convert external sentiment data to domain value objects
	sources, err := s.convertToSentimentSources(aggregated)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sentiment sources: %w", err)
	}
	
	// Create sentiment suggestion entity
	suggestion, err := entity.NewSentimentSuggestion(sources, aggregated.Reasoning, aggregated.Confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment suggestion: %w", err)
	}
	
	return &SentimentCollectionResult{
		Suggestion: suggestion,
		Sources:    sources,
		Reasoning:  aggregated.Reasoning,
		Confidence: aggregated.Confidence,
	}, nil
}

// QuickSentimentCheck performs lightweight analysis for monitoring
func (s *MarketSentimentService) QuickSentimentCheck() (*SentimentCollectionResult, error) {
	aggregated, err := s.aggregator.QuickAnalysis()
	if err != nil {
		return nil, fmt.Errorf("failed to perform quick sentiment check: %w", err)
	}
	
	sources, err := s.convertToSentimentSources(aggregated)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sentiment sources: %w", err)
	}
	
	suggestion, err := entity.NewSentimentSuggestion(sources, aggregated.Reasoning, aggregated.Confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment suggestion: %w", err)
	}
	
	return &SentimentCollectionResult{
		Suggestion: suggestion,
		Sources:    sources,
		Reasoning:  aggregated.Reasoning,
		Confidence: aggregated.Confidence,
	}, nil
}

// convertToSentimentSources converts external aggregated sentiment to domain value objects
func (s *MarketSentimentService) convertToSentimentSources(aggregated *external.AggregatedSentiment) (*vo.SentimentSources, error) {
	var fearGreedIndex int
	var newsScore, redditScore, socialScore float64
	
	// Extract Fear & Greed Index
	if aggregated.Sources.FearGreedIndex != nil {
		fearGreedIndex = aggregated.Sources.FearGreedIndex.Value
	}
	
	// Extract news score
	if aggregated.Sources.NewsAnalysis != nil {
		newsScore = aggregated.Sources.NewsAnalysis.OverallScore
	}
	
	// Extract Reddit score
	redditScore = aggregated.Sources.RedditScore
	
	// Calculate social score as average of non-traditional news sources
	socialScore = s.calculateSocialScore(aggregated)
	
	// Create domain value object
	sources, err := vo.NewSentimentSources(fearGreedIndex, newsScore, redditScore, socialScore)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment sources value object: %w", err)
	}
	
	return sources, nil
}

// calculateSocialScore computes social media sentiment from various sources
func (s *MarketSentimentService) calculateSocialScore(aggregated *external.AggregatedSentiment) float64 {
	if aggregated.Sources.NewsAnalysis == nil {
		return 0.0
	}
	
	socialSources := []string{"YouTubeChannels", "MastodonHashtags", "TelegramChannels"}
	var totalScore float64
	var count int
	
	for _, source := range socialSources {
		if sourceData, exists := aggregated.Sources.NewsAnalysis.SourceBreakdown[source]; exists {
			totalScore += sourceData.Score
			count++
		}
	}
	
	// If no specific social sources, use Reddit as proxy
	if count == 0 {
		return aggregated.Sources.RedditScore
	}
	
	return totalScore / float64(count)
}

// GetSentimentSuggestions generates trading suggestions based on sentiment level
func (s *MarketSentimentService) GetSentimentSuggestions(sentiment string) SentimentTradingSuggestions {
	suggestions := map[string]SentimentTradingSuggestions{
		"very_bullish": {
			TradeAmountMultiplier:    1.5,
			MinimumProfitThreshold:   0.8,
			IntervalSeconds:          300, // 5 minutes
			Recommendation:           "increase_exposure",
			ReasoningText:           "Market muito otimista - considere aumentar exposição",
		},
		"bullish": {
			TradeAmountMultiplier:    1.2,
			MinimumProfitThreshold:   1.0,
			IntervalSeconds:          600, // 10 minutes
			Recommendation:           "normal_plus",
			ReasoningText:           "Sentiment positivo - ligeiro aumento na agressividade",
		},
		"neutral": {
			TradeAmountMultiplier:    1.0,
			MinimumProfitThreshold:   1.5,
			IntervalSeconds:          900, // 15 minutes
			Recommendation:           "maintain",
			ReasoningText:           "Sentiment neutro - manter configurações atuais",
		},
		"bearish": {
			TradeAmountMultiplier:    0.7,
			MinimumProfitThreshold:   2.0,
			IntervalSeconds:          1800, // 30 minutes
			Recommendation:           "reduce_exposure",
			ReasoningText:           "Sentiment negativo - considere reduzir exposição",
		},
		"very_bearish": {
			TradeAmountMultiplier:    0.4,
			MinimumProfitThreshold:   3.0,
			IntervalSeconds:          3600, // 1 hour
			Recommendation:           "minimal_exposure",
			ReasoningText:           "Market muito pessimista - considere exposição mínima",
		},
	}
	
	if suggestion, exists := suggestions[sentiment]; exists {
		return suggestion
	}
	
	// Default to neutral if sentiment not recognized
	return suggestions["neutral"]
}

// SentimentTradingSuggestions contains trading parameter suggestions
type SentimentTradingSuggestions struct {
	TradeAmountMultiplier  float64 `json:"trade_amount_multiplier"`
	MinimumProfitThreshold float64 `json:"minimum_profit_threshold"`
	IntervalSeconds        int     `json:"interval_seconds"`
	Recommendation         string  `json:"recommendation"`
	ReasoningText          string  `json:"reasoning_text"`
}

// ScheduleRegularAnalysis can be called by a scheduler to perform regular sentiment analysis
func (s *MarketSentimentService) ScheduleRegularAnalysis() (*SentimentCollectionResult, error) {
	// Add timestamp logging for scheduled runs
	fmt.Printf("🔍 Starting scheduled sentiment analysis at %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	result, err := s.CollectMarketSentiment()
	if err != nil {
		fmt.Printf("❌ Scheduled sentiment analysis failed: %v\n", err)
		return nil, err
	}
	
	fmt.Printf("✅ Scheduled sentiment analysis completed. Sentiment: %s, Confidence: %.2f\n", 
		result.Suggestion.GetLevel(), result.Confidence)
	
	return result, nil
}

// ValidateDataSources checks if external data sources are accessible
func (s *MarketSentimentService) ValidateDataSources() error {
	// Quick validation by attempting to fetch Fear & Greed Index
	_, err := s.aggregator.QuickAnalysis()
	if err != nil {
		return fmt.Errorf("data source validation failed: %w", err)
	}
	
	return nil
}
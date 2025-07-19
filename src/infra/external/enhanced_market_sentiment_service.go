package external

import (
	"context"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
	"sync"
	"time"
)

type EnhancedMarketSentimentService struct {
	// Core components
	aggregator           *SentimentAggregator
	enhancedAnalyzer     *EnhancedSentimentAnalyzer
	parallelProcessor    *ParallelRSSProcessor
	healthMonitor        *RSSHealthMonitor
	cacheManager         *RSSCacheManager
	keywordManager       *DynamicKeywordManager
	
	// Configuration
	config               *EnhancedServiceConfig
	
	// Statistics
	stats                *ServiceStats
	statsMutex           sync.RWMutex
}

type EnhancedServiceConfig struct {
	// Processing configuration
	EnableParallelProcessing bool
	EnableIntelligentCaching bool
	EnableHealthMonitoring   bool
	EnableDynamicKeywords    bool
	
	// Performance tuning
	MaxWorkers              int
	CacheDefaultTTL         time.Duration
	HealthCheckInterval     time.Duration
	
	// Feature flags
	UseEnhancedNLP          bool
	EnableTrendingAnalysis  bool
	EnableMarketConditions  bool
	
	// Fallback configuration
	EnableAutoFallback      bool
	MinHealthyFeeds         int
	
	// Context timeouts
	DefaultTimeout          time.Duration
	HealthCheckTimeout      time.Duration
}

type ServiceStats struct {
	TotalAnalyses       int64     `json:"total_analyses"`
	SuccessfulAnalyses  int64     `json:"successful_analyses"`
	FailedAnalyses      int64     `json:"failed_analyses"`
	CacheHits           int64     `json:"cache_hits"`
	CacheMisses         int64     `json:"cache_misses"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastAnalysisTime    time.Time `json:"last_analysis_time"`
	HealthCheckCount    int64     `json:"health_check_count"`
	TrendingKeywords    []string  `json:"trending_keywords"`
}

type EnhancedSentimentResult struct {
	// Original result data
	Suggestion *entity.SentimentSuggestion
	Sources    *vo.SentimentSources
	Reasoning  string
	Confidence float64
	
	// Enhanced analysis data
	TrendingKeywords    []TrendingKeyword      `json:"trending_keywords,omitempty"`
	MarketConditions    map[string]MarketCondition `json:"market_conditions,omitempty"`
	ProcessingStats     *BatchProcessingResult `json:"processing_stats,omitempty"`
	HealthSummary       *HealthSummary         `json:"health_summary,omitempty"`
	KeywordInsights     *KeywordLearningResult `json:"keyword_insights,omitempty"`
	
	// Performance metrics
	ProcessingTime      time.Duration          `json:"processing_time"`
	DataFreshness       time.Duration          `json:"data_freshness"`
	SourceReliability   map[string]float64     `json:"source_reliability,omitempty"`
}

func NewEnhancedMarketSentimentService(config *EnhancedServiceConfig) *EnhancedMarketSentimentService {
	if config == nil {
		config = &EnhancedServiceConfig{
			EnableParallelProcessing: true,
			EnableIntelligentCaching: true,
			EnableHealthMonitoring:   true,
			EnableDynamicKeywords:    true,
			MaxWorkers:              8,
			CacheDefaultTTL:         15 * time.Minute,
			HealthCheckInterval:     5 * time.Minute,
			UseEnhancedNLP:          true,
			EnableTrendingAnalysis:  true,
			EnableMarketConditions:  true,
			EnableAutoFallback:      true,
			MinHealthyFeeds:         3,
			DefaultTimeout:          30 * time.Second,
			HealthCheckTimeout:      10 * time.Second,
		}
	}
	
	service := &EnhancedMarketSentimentService{
		aggregator: NewSentimentAggregator(),
		config:     config,
		stats:      &ServiceStats{},
	}
	
	// Initialize enhanced components based on configuration
	if config.UseEnhancedNLP {
		service.enhancedAnalyzer = NewEnhancedSentimentAnalyzer()
	}
	
	if config.EnableIntelligentCaching {
		cacheConfig := CacheConfig{
			DefaultTTL:      config.CacheDefaultTTL,
			MaxCacheSize:    1000,
			CleanupInterval: 5 * time.Minute,
		}
		service.cacheManager = NewRSSCacheManager(cacheConfig)
	}
	
	if config.EnableHealthMonitoring {
		service.healthMonitor = NewRSSHealthMonitor()
		service.healthMonitor.SetCheckInterval(config.HealthCheckInterval)
		service.healthMonitor.StartMonitoring()
	}
	
	if config.EnableParallelProcessing {
		feedReader := NewRSSFeedReader()
		processingConfig := ProcessingConfig{
			MaxWorkers: config.MaxWorkers,
			Timeout:    config.DefaultTimeout,
		}
		
		var analyzer SentimentAnalyzerInterface
		if service.enhancedAnalyzer != nil {
			analyzer = &enhancedAnalyzerAdapter{service.enhancedAnalyzer}
		} else {
			analyzer = &basicAnalyzerAdapter{NewSentimentAnalyzer()}
		}
		
		service.parallelProcessor = NewParallelRSSProcessor(
			feedReader,
			service.healthMonitor,
			service.cacheManager,
			analyzer,
			processingConfig,
		)
	}
	
	if config.EnableDynamicKeywords {
		service.keywordManager = NewDynamicKeywordManager()
	}
	
	return service
}

// Enhanced sentiment collection with all features
func (s *EnhancedMarketSentimentService) CollectMarketSentiment() (*EnhancedSentimentResult, error) {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), s.config.DefaultTimeout)
	defer cancel()
	
	s.updateStats(func(stats *ServiceStats) {
		stats.TotalAnalyses++
		stats.LastAnalysisTime = startTime
	})
	
	// Use parallel processing if enabled
	if s.config.EnableParallelProcessing && s.parallelProcessor != nil {
		return s.collectWithParallelProcessing(ctx, startTime)
	}
	
	// Fallback to original method
	return s.collectWithOriginalMethod(ctx, startTime)
}

func (s *EnhancedMarketSentimentService) collectWithParallelProcessing(ctx context.Context, startTime time.Time) (*EnhancedSentimentResult, error) {
	processingConfig := ProcessingConfig{
		UseCache:          s.config.EnableIntelligentCaching,
		ParallelSentiment: s.config.UseEnhancedNLP,
		MaxWorkers:        s.config.MaxWorkers,
		Priority: map[string]int{
			"CoinDesk":      10,
			"CoinTelegraph": 9,
			"RedditCrypto":  8,
			"BitcoinCom":    7,
			"Decrypt":       6,
		},
	}
	
	// Use fallback processing if health monitoring detects issues
	if s.config.EnableAutoFallback && s.healthMonitor != nil {
		batchResult := s.parallelProcessor.ProcessWithFallback(ctx, processingConfig)
		return s.processParallelResults(batchResult, startTime)
	}
	
	// Normal parallel processing
	batchResult := s.parallelProcessor.ProcessAllFeedsParallel(ctx, processingConfig)
	return s.processParallelResults(batchResult, startTime)
}

func (s *EnhancedMarketSentimentService) processParallelResults(batchResult BatchProcessingResult, startTime time.Time) (*EnhancedSentimentResult, error) {
	// Combine all news items from successful feeds
	var allNewsItems []NewsItem
	var sentimentResults []NewsAnalysisResult
	
	for _, result := range batchResult.Results {
		if result.Error == nil {
			allNewsItems = append(allNewsItems, result.NewsItems...)
			if result.SentimentData != nil {
				sentimentResults = append(sentimentResults, *result.SentimentData)
			}
		}
	}
	
	if len(allNewsItems) == 0 {
		s.updateStats(func(stats *ServiceStats) { stats.FailedAnalyses++ })
		return nil, fmt.Errorf("no news items collected from any feed")
	}
	
	// Perform enhanced sentiment analysis
	var aggregatedSentiment *AggregatedSentiment
	var err error
	
	if s.config.UseEnhancedNLP && s.enhancedAnalyzer != nil {
		aggregatedSentiment, err = s.enhancedSentimentAnalysis(allNewsItems)
	} else {
		// Fallback to original aggregator
		aggregatedSentiment, err = s.aggregator.CollectAndAnalyze()
	}
	
	if err != nil {
		s.updateStats(func(stats *ServiceStats) { stats.FailedAnalyses++ })
		return nil, fmt.Errorf("sentiment analysis failed: %w", err)
	}
	
	// Convert to domain objects
	sources, err := s.convertToSentimentSources(aggregatedSentiment)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sentiment sources: %w", err)
	}
	
	suggestion, err := entity.NewSentimentSuggestion(sources, aggregatedSentiment.Reasoning, aggregatedSentiment.Confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment suggestion: %w", err)
	}
	
	// Build enhanced result
	result := &EnhancedSentimentResult{
		Suggestion:      suggestion,
		Sources:         sources,
		Reasoning:       aggregatedSentiment.Reasoning,
		Confidence:      aggregatedSentiment.Confidence,
		ProcessingTime:  time.Since(startTime),
		ProcessingStats: &batchResult,
	}
	
	// Add enhanced features if enabled
	if s.config.EnableTrendingAnalysis && s.keywordManager != nil {
		result.TrendingKeywords = s.keywordManager.GetTrendingKeywords(10)
	}
	
	if s.config.EnableMarketConditions && s.keywordManager != nil {
		result.MarketConditions = s.keywordManager.GetMarketConditions()
	}
	
	if s.config.EnableHealthMonitoring && s.healthMonitor != nil {
		healthSummary := s.healthMonitor.GetHealthSummary()
		result.HealthSummary = &healthSummary
	}
	
	// Update keyword learning if enabled
	if s.config.EnableDynamicKeywords && s.keywordManager != nil {
		for _, item := range allNewsItems {
			learningResult := s.keywordManager.LearnFromText(
				item.Content,
				aggregatedSentiment.OverallScore,
				0.0, // Would need market price data for correlation
			)
			result.KeywordInsights = &learningResult
		}
	}
	
	// Calculate source reliability
	result.SourceReliability = s.calculateSourceReliability(batchResult)
	
	// Update cache hit statistics
	cacheHits := int64(batchResult.CacheHitCount)
	cacheMisses := int64(batchResult.SuccessCount - batchResult.CacheHitCount)
	
	s.updateStats(func(stats *ServiceStats) {
		stats.SuccessfulAnalyses++
		stats.CacheHits += cacheHits
		stats.CacheMisses += cacheMisses
		stats.AverageResponseTime = (stats.AverageResponseTime + result.ProcessingTime) / 2
		
		if len(result.TrendingKeywords) > 0 {
			stats.TrendingKeywords = make([]string, len(result.TrendingKeywords))
			for i, kw := range result.TrendingKeywords {
				stats.TrendingKeywords[i] = kw.Word
			}
		}
	})
	
	return result, nil
}

func (s *EnhancedMarketSentimentService) collectWithOriginalMethod(ctx context.Context, startTime time.Time) (*EnhancedSentimentResult, error) {
	// Original aggregator method
	aggregated, err := s.aggregator.CollectAndAnalyze()
	if err != nil {
		s.updateStats(func(stats *ServiceStats) { stats.FailedAnalyses++ })
		return nil, fmt.Errorf("failed to collect sentiment data: %w", err)
	}
	
	sources, err := s.convertToSentimentSources(aggregated)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sentiment sources: %w", err)
	}
	
	suggestion, err := entity.NewSentimentSuggestion(sources, aggregated.Reasoning, aggregated.Confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment suggestion: %w", err)
	}
	
	result := &EnhancedSentimentResult{
		Suggestion:     suggestion,
		Sources:        sources,
		Reasoning:      aggregated.Reasoning,
		Confidence:     aggregated.Confidence,
		ProcessingTime: time.Since(startTime),
	}
	
	s.updateStats(func(stats *ServiceStats) {
		stats.SuccessfulAnalyses++
		stats.AverageResponseTime = (stats.AverageResponseTime + result.ProcessingTime) / 2
	})
	
	return result, nil
}

// Quick sentiment check with enhanced features
func (s *EnhancedMarketSentimentService) QuickSentimentCheck() (*EnhancedSentimentResult, error) {
	startTime := time.Now()
	
	// Use cache for quick checks if available
	if s.config.EnableIntelligentCaching && s.cacheManager != nil {
		if cachedNews, hit := s.cacheManager.GetCachedRecentNews(1); hit {
			return s.processQuickCheck(cachedNews, startTime, true)
		}
	}
	
	// Fallback to original quick analysis
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
	
	return &EnhancedSentimentResult{
		Suggestion:     suggestion,
		Sources:        sources,
		Reasoning:      aggregated.Reasoning,
		Confidence:     aggregated.Confidence,
		ProcessingTime: time.Since(startTime),
	}, nil
}

// Validate data sources with enhanced health monitoring
func (s *EnhancedMarketSentimentService) ValidateDataSources() error {
	if s.config.EnableHealthMonitoring && s.healthMonitor != nil {
		return s.healthMonitor.ValidateAllFeeds()
	}
	
	// Fallback to original validation
	_, err := s.aggregator.QuickAnalysis()
	if err != nil {
		return fmt.Errorf("data source validation failed: %w", err)
	}
	
	return nil
}

// Get comprehensive service statistics
func (s *EnhancedMarketSentimentService) GetServiceStats() ServiceStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := *s.stats
	
	// Add health check count if monitoring is enabled
	if s.healthMonitor != nil {
		s.updateStats(func(stats *ServiceStats) {
			stats.HealthCheckCount++
		})
	}
	
	return statsCopy
}

// Shutdown gracefully stops all background services
func (s *EnhancedMarketSentimentService) Shutdown() {
	if s.healthMonitor != nil {
		s.healthMonitor.StopMonitoring()
	}
	
	if s.cacheManager != nil {
		s.cacheManager.Stop()
	}
}

// Private helper methods

func (s *EnhancedMarketSentimentService) enhancedSentimentAnalysis(newsItems []NewsItem) (*AggregatedSentiment, error) {
	// This would integrate with the enhanced analyzer
	// For now, fallback to original aggregator
	return s.aggregator.CollectAndAnalyze()
}

func (s *EnhancedMarketSentimentService) convertToSentimentSources(aggregated *AggregatedSentiment) (*vo.SentimentSources, error) {
	var fearGreedIndex int
	var newsScore, redditScore, socialScore float64
	
	if aggregated.Sources.FearGreedIndex != nil {
		fearGreedIndex = aggregated.Sources.FearGreedIndex.Value
	}
	
	if aggregated.Sources.NewsAnalysis != nil {
		newsScore = aggregated.Sources.NewsAnalysis.OverallScore
	}
	
	redditScore = aggregated.Sources.RedditScore
	socialScore = s.calculateSocialScore(aggregated)
	
	return vo.NewSentimentSources(fearGreedIndex, newsScore, redditScore, socialScore)
}

func (s *EnhancedMarketSentimentService) calculateSocialScore(aggregated *AggregatedSentiment) float64 {
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
	
	if count == 0 {
		return aggregated.Sources.RedditScore
	}
	
	return totalScore / float64(count)
}

func (s *EnhancedMarketSentimentService) processQuickCheck(newsItems []NewsItem, startTime time.Time, fromCache bool) (*EnhancedSentimentResult, error) {
	// Process cached news items for quick analysis
	// This is a simplified implementation
	return nil, fmt.Errorf("quick check processing not implemented")
}

func (s *EnhancedMarketSentimentService) calculateSourceReliability(batchResult BatchProcessingResult) map[string]float64 {
	reliability := make(map[string]float64)
	
	for _, result := range batchResult.Results {
		if result.Error == nil {
			// Calculate reliability based on response time and item count
			responseScore := 1.0
			if result.ProcessingTime > 10*time.Second {
				responseScore = 0.5
			} else if result.ProcessingTime > 5*time.Second {
				responseScore = 0.8
			}
			
			itemScore := 1.0
			if len(result.NewsItems) < 5 {
				itemScore = 0.7
			} else if len(result.NewsItems) > 20 {
				itemScore = 0.9
			}
			
			reliability[result.Source] = (responseScore + itemScore) / 2
		} else {
			reliability[result.Source] = 0.0
		}
	}
	
	return reliability
}

func (s *EnhancedMarketSentimentService) updateStats(fn func(*ServiceStats)) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	fn(s.stats)
}

// Adapter interfaces for compatibility

type enhancedAnalyzerAdapter struct {
	analyzer *EnhancedSentimentAnalyzer
}

func (a *enhancedAnalyzerAdapter) AnalyzeNews(newsItems []NewsItem) NewsAnalysisResult {
	// Convert enhanced analysis to standard result
	// This is a simplified adapter
	return NewsAnalysisResult{
		TotalArticles: len(newsItems),
		OverallScore:  0.0,
	}
}

type basicAnalyzerAdapter struct {
	analyzer *SentimentAnalyzer
}

func (a *basicAnalyzerAdapter) AnalyzeNews(newsItems []NewsItem) NewsAnalysisResult {
	return a.analyzer.AnalyzeNews(newsItems)
}
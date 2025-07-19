package external

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

type LLMSentimentAnalyzer struct {
	openaiClient    *OpenAIClient
	cacheManager    *LLMCache
	fallbackAnalyzer *SentimentAnalyzer
	config          *LLMAnalyzerConfig
	stats           *LLMAnalyzerStats
	statsMutex      sync.RWMutex
}

type LLMAnalyzerConfig struct {
	EnableLLM       bool
	EnableCache     bool
	EnableFallback  bool
	CacheTTL        time.Duration
	MaxConcurrency  int
	Timeout         time.Duration
	CostLimit       float64 // Daily cost limit in USD
}

type LLMAnalyzerStats struct {
	TotalAnalyses     int64         `json:"total_analyses"`
	LLMAnalyses       int64         `json:"llm_analyses"`
	FallbackAnalyses  int64         `json:"fallback_analyses"`
	CacheHits         int64         `json:"cache_hits"`
	SuccessRate       float64       `json:"success_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	DailyCost         float64       `json:"daily_cost_usd"`
	LastResetTime     time.Time     `json:"last_reset_time"`
}

type LLMCache struct {
	entries map[string]*LLMCacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
}

type LLMCacheEntry struct {
	Result    *LLMAnalysisResult
	CreatedAt time.Time
	ExpiresAt time.Time
	AccessCount int
}

type EnhancedNewsAnalysisResult struct {
	OverallScore        float64                      `json:"overall_score"`
	TotalArticles       int                          `json:"total_articles"`
	PositiveArticles    int                          `json:"positive_articles"`
	NegativeArticles    int                          `json:"negative_articles"`
	NeutralArticles     int                          `json:"neutral_articles"`
	SourceBreakdown     map[string]SentimentResult   `json:"source_breakdown"`
	LLMAnalysisResults  []LLMAnalysisResult          `json:"llm_analysis_results,omitempty"`
	ProcessingMethod    string                       `json:"processing_method"` // "llm", "fallback", "hybrid"
	AnalysisQuality     string                       `json:"analysis_quality"`  // "high", "medium", "low"
	KeyInsights         []string                     `json:"key_insights,omitempty"`
	MarketContext       string                       `json:"market_context,omitempty"`
}

func NewLLMSentimentAnalyzer() *LLMSentimentAnalyzer {
	config := &LLMAnalyzerConfig{
		EnableLLM:       getEnvBoolOrDefault("USE_LLM_ANALYSIS", true),
		EnableCache:     true,
		EnableFallback:  true,
		CacheTTL:        30 * time.Minute,
		MaxConcurrency:  5,
		Timeout:         90 * time.Second,
		CostLimit:       10.0, // $10/day default limit
	}

	analyzer := &LLMSentimentAnalyzer{
		config:           config,
		fallbackAnalyzer: NewSentimentAnalyzer(),
		stats: &LLMAnalyzerStats{
			LastResetTime: time.Now(),
		},
	}

	// Initialize OpenAI client if LLM is enabled
	if config.EnableLLM {
		openaiConfig := OpenAIConfig{
			Timeout:     config.Timeout,
			MaxRequests: 60,
			TimeWindow:  time.Minute,
		}
		analyzer.openaiClient = NewOpenAIClient(openaiConfig)
	}

	// Initialize cache if enabled
	if config.EnableCache {
		analyzer.cacheManager = &LLMCache{
			entries: make(map[string]*LLMCacheEntry),
			ttl:     config.CacheTTL,
		}
	}

	return analyzer
}

// AnalyzeNews processes news items with LLM analysis and fallback
func (l *LLMSentimentAnalyzer) AnalyzeNews(newsItems []NewsItem) EnhancedNewsAnalysisResult {
	startTime := time.Now()
	
	l.updateStats(func(stats *LLMAnalyzerStats) {
		stats.TotalAnalyses++
	})

	result := EnhancedNewsAnalysisResult{
		TotalArticles:    len(newsItems),
		SourceBreakdown:  make(map[string]SentimentResult),
		ProcessingMethod: "fallback", // Default
		AnalysisQuality:  "medium",   // Default
	}

	// Check if we should reset daily stats
	l.checkDailyReset()

	// Determine processing method
	useLLM := l.shouldUseLLM()
	
	if useLLM && l.openaiClient != nil && l.openaiClient.IsConfigured() {
		llmResult, err := l.analyzewithLLM(newsItems)
		if err != nil {
			fmt.Printf("🔄 LLM analysis failed, falling back to keyword analysis: %v\n", err)
			result = l.analyzeWithFallback(newsItems)
			result.ProcessingMethod = "fallback"
			result.AnalysisQuality = "medium"
		} else {
			result = *llmResult
			result.ProcessingMethod = "llm"
			result.AnalysisQuality = "high"
			
			l.updateStats(func(stats *LLMAnalyzerStats) {
				stats.LLMAnalyses++
			})
		}
	} else {
		result = l.analyzeWithFallback(newsItems)
		result.ProcessingMethod = "fallback"
		result.AnalysisQuality = "medium"
	}

	// Update statistics
	responseTime := time.Since(startTime)
	l.updateStats(func(stats *LLMAnalyzerStats) {
		stats.AverageResponseTime = (stats.AverageResponseTime + responseTime) / 2
		if stats.TotalAnalyses > 0 {
			stats.SuccessRate = float64(stats.LLMAnalyses+stats.FallbackAnalyses) / float64(stats.TotalAnalyses)
		}
	})

	return result
}

func (l *LLMSentimentAnalyzer) analyzewithLLM(newsItems []NewsItem) (*EnhancedNewsAnalysisResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.config.Timeout)
	defer cancel()

	result := &EnhancedNewsAnalysisResult{
		TotalArticles:      len(newsItems),
		SourceBreakdown:    make(map[string]SentimentResult),
		LLMAnalysisResults: make([]LLMAnalysisResult, 0, len(newsItems)),
		KeyInsights:        make([]string, 0),
	}

	var totalScore float64
	sourceScores := make(map[string][]float64)
	var keyInsights []string

	// Process articles with optional caching
	for _, item := range newsItems {
		var llmResult *LLMAnalysisResult
		var err error

		// Check cache first
		if l.config.EnableCache {
			if cached := l.getCachedResult(item.Content); cached != nil {
				llmResult = cached
				l.updateStats(func(stats *LLMAnalyzerStats) {
					stats.CacheHits++
				})
			}
		}

		// Analyze with LLM if not cached
		if llmResult == nil {
			llmResult, err = l.openaiClient.AnalyzeSentiment(ctx, item.Content)
			if err != nil {
				return nil, fmt.Errorf("LLM analysis failed for article from %s: %w", item.Source, err)
			}

			// Cache the result
			if l.config.EnableCache {
				l.cacheResult(item.Content, llmResult)
			}
		}

		// Accumulate results
		totalScore += llmResult.Score
		result.LLMAnalysisResults = append(result.LLMAnalysisResults, *llmResult)

		// Track by source
		if _, exists := sourceScores[item.Source]; !exists {
			sourceScores[item.Source] = []float64{}
		}
		sourceScores[item.Source] = append(sourceScores[item.Source], llmResult.Score)

		// Collect insights
		if llmResult.Reasoning != "" {
			keyInsights = append(keyInsights, llmResult.Reasoning)
		}

		// Classify article
		if llmResult.Score > 0.1 {
			result.PositiveArticles++
		} else if llmResult.Score < -0.1 {
			result.NegativeArticles++
		} else {
			result.NeutralArticles++
		}
	}

	// Calculate overall score
	if len(newsItems) > 0 {
		result.OverallScore = totalScore / float64(len(newsItems))
	}

	// Generate source breakdown
	for source, scores := range sourceScores {
		var sourceTotal float64
		var posCount, negCount, neutCount int

		for _, score := range scores {
			sourceTotal += score
			if score > 0.1 {
				posCount++
			} else if score < -0.1 {
				negCount++
			} else {
				neutCount++
			}
		}

		avgScore := sourceTotal / float64(len(scores))
		classification := "neutral"
		if avgScore > 0.1 {
			classification = "positive"
		} else if avgScore < -0.1 {
			classification = "negative"
		}

		result.SourceBreakdown[source] = SentimentResult{
			Score:          avgScore,
			PositiveCount:  posCount,
			NegativeCount:  negCount,
			NeutralCount:   neutCount,
			Classification: classification,
			Confidence:     0.9, // High confidence for LLM analysis
		}
	}

	// Limit insights to top 5 most relevant
	if len(keyInsights) > 5 {
		keyInsights = keyInsights[:5]
	}
	result.KeyInsights = keyInsights

	// Generate market context
	result.MarketContext = l.generateMarketContext(result.OverallScore, result.PositiveArticles, result.NegativeArticles)

	return result, nil
}

func (l *LLMSentimentAnalyzer) analyzeWithFallback(newsItems []NewsItem) EnhancedNewsAnalysisResult {
	l.updateStats(func(stats *LLMAnalyzerStats) {
		stats.FallbackAnalyses++
	})

	// Use the existing basic sentiment analyzer
	basicResult := l.fallbackAnalyzer.AnalyzeNews(newsItems)

	// Convert to enhanced result format
	return EnhancedNewsAnalysisResult{
		OverallScore:     basicResult.OverallScore,
		TotalArticles:    basicResult.TotalArticles,
		PositiveArticles: basicResult.PositiveArticles,
		NegativeArticles: basicResult.NegativeArticles,
		NeutralArticles:  basicResult.NeutralArticles,
		SourceBreakdown:  basicResult.SourceBreakdown,
		ProcessingMethod: "fallback",
		AnalysisQuality:  "medium",
		KeyInsights:      []string{"Keyword-based analysis performed"},
		MarketContext:    l.generateMarketContext(basicResult.OverallScore, basicResult.PositiveArticles, basicResult.NegativeArticles),
	}
}

func (l *LLMSentimentAnalyzer) shouldUseLLM() bool {
	if !l.config.EnableLLM {
		return false
	}

	// Check daily cost limit
	l.statsMutex.RLock()
	dailyCost := l.stats.DailyCost
	l.statsMutex.RUnlock()

	if dailyCost >= l.config.CostLimit {
		fmt.Printf("💰 Daily LLM cost limit reached ($%.2f), using fallback analysis\n", dailyCost)
		return false
	}

	return true
}

func (l *LLMSentimentAnalyzer) generateMarketContext(score float64, positive, negative int) string {
	if score > 0.5 {
		return fmt.Sprintf("Strong bullish sentiment detected across %d positive articles", positive)
	} else if score > 0.2 {
		return fmt.Sprintf("Moderately positive sentiment with %d bullish vs %d bearish articles", positive, negative)
	} else if score < -0.5 {
		return fmt.Sprintf("Strong bearish sentiment detected across %d negative articles", negative)
	} else if score < -0.2 {
		return fmt.Sprintf("Moderately negative sentiment with %d bearish vs %d bullish articles", negative, positive)
	}
	return fmt.Sprintf("Mixed market sentiment with %d positive and %d negative articles", positive, negative)
}

// Cache management methods
func (l *LLMSentimentAnalyzer) getCachedResult(content string) *LLMAnalysisResult {
	if l.cacheManager == nil {
		return nil
	}

	l.cacheManager.mutex.RLock()
	defer l.cacheManager.mutex.RUnlock()

	key := l.generateCacheKey(content)
	entry, exists := l.cacheManager.entries[key]
	if !exists {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	entry.AccessCount++
	return entry.Result
}

func (l *LLMSentimentAnalyzer) cacheResult(content string, result *LLMAnalysisResult) {
	if l.cacheManager == nil {
		return
	}

	l.cacheManager.mutex.Lock()
	defer l.cacheManager.mutex.Unlock()

	key := l.generateCacheKey(content)
	l.cacheManager.entries[key] = &LLMCacheEntry{
		Result:      result,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(l.cacheManager.ttl),
		AccessCount: 1,
	}

	// Clean expired entries periodically
	l.cleanExpiredCache()
}

func (l *LLMSentimentAnalyzer) generateCacheKey(content string) string {
	// Simple hash of content for cache key
	if len(content) > 100 {
		return fmt.Sprintf("llm_%x", content[:100])
	}
	return fmt.Sprintf("llm_%x", content)
}

func (l *LLMSentimentAnalyzer) cleanExpiredCache() {
	now := time.Now()
	for key, entry := range l.cacheManager.entries {
		if now.After(entry.ExpiresAt) {
			delete(l.cacheManager.entries, key)
		}
	}
}

func (l *LLMSentimentAnalyzer) checkDailyReset() {
	l.statsMutex.RLock()
	lastReset := l.stats.LastResetTime
	l.statsMutex.RUnlock()

	if time.Since(lastReset) > 24*time.Hour {
		l.updateStats(func(stats *LLMAnalyzerStats) {
			stats.DailyCost = 0
			stats.LastResetTime = time.Now()
		})
	}
}

func (l *LLMSentimentAnalyzer) updateStats(fn func(*LLMAnalyzerStats)) {
	l.statsMutex.Lock()
	defer l.statsMutex.Unlock()
	fn(l.stats)
}

func (l *LLMSentimentAnalyzer) GetStats() LLMAnalyzerStats {
	l.statsMutex.RLock()
	defer l.statsMutex.RUnlock()
	return *l.stats
}

func (l *LLMSentimentAnalyzer) GetOpenAIStats() *OpenAIStats {
	if l.openaiClient == nil {
		return nil
	}
	stats := l.openaiClient.GetStats()
	
	// Update our daily cost from OpenAI stats
	l.updateStats(func(myStats *LLMAnalyzerStats) {
		myStats.DailyCost = stats.EstimatedCost
	})
	
	return &stats
}

// Helper function
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// Interface compliance check - ensure we can be used as a drop-in replacement
func (l *LLMSentimentAnalyzer) AnalyzeText(text string) SentimentResult {
	// Create a mock NewsItem for compatibility
	newsItem := NewsItem{Content: text}
	result := l.AnalyzeNews([]NewsItem{newsItem})
	
	// Convert back to SentimentResult for backward compatibility
	return SentimentResult{
		Score:          result.OverallScore,
		PositiveCount:  result.PositiveArticles,
		NegativeCount:  result.NegativeArticles,
		NeutralCount:   result.NeutralArticles,
		Classification: l.scoreToClassification(result.OverallScore),
		Confidence:     0.8, // Default confidence
		AnalyzedText:   text,
	}
}

func (l *LLMSentimentAnalyzer) scoreToClassification(score float64) string {
	if score > 0.1 {
		return "positive"
	} else if score < -0.1 {
		return "negative"
	}
	return "neutral"
}
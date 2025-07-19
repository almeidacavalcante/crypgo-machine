package external

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ParallelRSSProcessor struct {
	feedReader        *RSSFeedReader
	healthMonitor     *RSSHealthMonitor
	cacheManager      *RSSCacheManager
	sentimentAnalyzer SentimentAnalyzerInterface
	maxWorkers        int
	timeout           time.Duration
}

type SentimentAnalyzerInterface interface {
	AnalyzeNews(newsItems []NewsItem) NewsAnalysisResult
}

type FeedProcessingJob struct {
	Source   string
	URL      string
	Priority int // Higher number = higher priority
}

type FeedProcessingResult struct {
	Source        string
	NewsItems     []NewsItem
	SentimentData *NewsAnalysisResult
	Error         error
	ProcessingTime time.Duration
	CacheHit      bool
}

type BatchProcessingResult struct {
	Results        []FeedProcessingResult
	TotalItems     int
	SuccessCount   int
	ErrorCount     int
	CacheHitCount  int
	TotalTime      time.Duration
	AverageSentiment float64
}

type ProcessingConfig struct {
	MaxWorkers       int
	Timeout          time.Duration
	UseCache         bool
	ParallelSentiment bool
	Priority         map[string]int // Source -> priority mapping
}

func NewParallelRSSProcessor(
	feedReader *RSSFeedReader,
	healthMonitor *RSSHealthMonitor,
	cacheManager *RSSCacheManager,
	sentimentAnalyzer SentimentAnalyzerInterface,
	config ProcessingConfig,
) *ParallelRSSProcessor {
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &ParallelRSSProcessor{
		feedReader:        feedReader,
		healthMonitor:     healthMonitor,
		cacheManager:      cacheManager,
		sentimentAnalyzer: sentimentAnalyzer,
		maxWorkers:        config.MaxWorkers,
		timeout:           config.Timeout,
	}
}

// ProcessAllFeedsParallel processes all RSS feeds concurrently with intelligent caching and error handling
func (p *ParallelRSSProcessor) ProcessAllFeedsParallel(ctx context.Context, config ProcessingConfig) BatchProcessingResult {
	startTime := time.Now()
	
	// Create jobs for all feeds
	jobs := p.createProcessingJobs(config.Priority)
	
	// Create channels for communication
	jobChan := make(chan FeedProcessingJob, len(jobs))
	resultChan := make(chan FeedProcessingResult, len(jobs))
	
	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < p.maxWorkers && i < len(jobs); i++ {
		wg.Add(1)
		go p.worker(ctx, jobChan, resultChan, &wg, config)
	}
	
	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Process results
	var results []FeedProcessingResult
	for result := range resultChan {
		results = append(results, result)
	}
	
	return p.aggregateResults(results, time.Since(startTime))
}

// ProcessPrioritizedFeeds processes feeds with priority-based ordering
func (p *ParallelRSSProcessor) ProcessPrioritizedFeeds(ctx context.Context, sources []string, config ProcessingConfig) BatchProcessingResult {
	startTime := time.Now()
	
	// Create jobs for specified sources only
	jobs := []FeedProcessingJob{}
	for _, source := range sources {
		if url, exists := CryptoRSSFeeds[source]; exists {
			priority := 1
			if config.Priority != nil {
				if p, exists := config.Priority[source]; exists {
					priority = p
				}
			}
			
			jobs = append(jobs, FeedProcessingJob{
				Source:   source,
				URL:      url,
				Priority: priority,
			})
		}
	}
	
	// Sort jobs by priority (higher priority first)
	p.sortJobsByPriority(jobs)
	
	// Process jobs
	results := p.processJobsBatch(ctx, jobs, config)
	
	return p.aggregateResults(results, time.Since(startTime))
}

// ProcessWithFallback processes feeds with automatic fallback to healthy sources
func (p *ParallelRSSProcessor) ProcessWithFallback(ctx context.Context, config ProcessingConfig) BatchProcessingResult {
	// Get health status for all feeds
	healthSummary := p.healthMonitor.GetHealthSummary()
	
	// Create jobs prioritizing healthy feeds
	jobs := []FeedProcessingJob{}
	for source, url := range CryptoRSSFeeds {
		priority := 1
		
		// Boost priority for healthy feeds
		for _, status := range healthSummary.FeedStatuses {
			if status.Source == source && status.IsHealthy {
				priority += 10
				break
			}
		}
		
		// Apply user-defined priorities
		if config.Priority != nil {
			if p, exists := config.Priority[source]; exists {
				priority += p
			}
		}
		
		jobs = append(jobs, FeedProcessingJob{
			Source:   source,
			URL:      url,
			Priority: priority,
		})
	}
	
	p.sortJobsByPriority(jobs)
	
	startTime := time.Now()
	results := p.processJobsBatch(ctx, jobs, config)
	
	return p.aggregateResults(results, time.Since(startTime))
}

// worker processes feed jobs concurrently
func (p *ParallelRSSProcessor) worker(
	ctx context.Context,
	jobChan <-chan FeedProcessingJob,
	resultChan chan<- FeedProcessingResult,
	wg *sync.WaitGroup,
	config ProcessingConfig,
) {
	defer wg.Done()
	
	for {
		select {
		case job, ok := <-jobChan:
			if !ok {
				return
			}
			
			result := p.processJob(ctx, job, config)
			
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
			
		case <-ctx.Done():
			return
		}
	}
}

// processJob handles a single feed processing job
func (p *ParallelRSSProcessor) processJob(ctx context.Context, job FeedProcessingJob, config ProcessingConfig) FeedProcessingResult {
	startTime := time.Now()
	
	result := FeedProcessingResult{
		Source: job.Source,
	}
	
	// Check cache first if enabled
	if config.UseCache && p.cacheManager != nil {
		if cachedNews, hit := p.cacheManager.GetCachedFeed(job.Source); hit {
			result.NewsItems = cachedNews
			result.CacheHit = true
			
			// Try to get cached sentiment too
			if cachedSentiment, sentHit := p.cacheManager.GetCachedSentiment(job.Source); sentHit {
				result.SentimentData = cachedSentiment
			}
			
			result.ProcessingTime = time.Since(startTime)
			return result
		}
	}
	
	// Create context with timeout for this job
	jobCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	
	// Fetch feed with timeout
	newsItems, err := p.fetchFeedWithTimeout(jobCtx, job.URL, job.Source)
	if err != nil {
		result.Error = err
		result.ProcessingTime = time.Since(startTime)
		return result
	}
	
	result.NewsItems = newsItems
	
	// Cache the news items if caching is enabled
	if config.UseCache && p.cacheManager != nil {
		p.cacheManager.CacheFeed(job.Source, newsItems)
	}
	
	// Perform sentiment analysis if configured and analyzer is available
	if config.ParallelSentiment && p.sentimentAnalyzer != nil && len(newsItems) > 0 {
		sentimentResult := p.sentimentAnalyzer.AnalyzeNews(newsItems)
		result.SentimentData = &sentimentResult
		
		// Cache sentiment if caching is enabled
		if config.UseCache && p.cacheManager != nil {
			p.cacheManager.CacheSentiment(job.Source, &sentimentResult)
		}
	}
	
	result.ProcessingTime = time.Since(startTime)
	return result
}

// fetchFeedWithTimeout fetches a feed with context timeout
func (p *ParallelRSSProcessor) fetchFeedWithTimeout(ctx context.Context, url, source string) ([]NewsItem, error) {
	type result struct {
		items []NewsItem
		err   error
	}
	
	resultChan := make(chan result, 1)
	
	go func() {
		items, err := p.feedReader.FetchFeed(url, source)
		resultChan <- result{items: items, err: err}
	}()
	
	select {
	case res := <-resultChan:
		return res.items, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("feed fetch timeout for %s: %w", source, ctx.Err())
	}
}

// processJobsBatch processes a batch of jobs with the worker pool
func (p *ParallelRSSProcessor) processJobsBatch(ctx context.Context, jobs []FeedProcessingJob, config ProcessingConfig) []FeedProcessingResult {
	if len(jobs) == 0 {
		return []FeedProcessingResult{}
	}
	
	jobChan := make(chan FeedProcessingJob, len(jobs))
	resultChan := make(chan FeedProcessingResult, len(jobs))
	
	var wg sync.WaitGroup
	workerCount := p.maxWorkers
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}
	
	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go p.worker(ctx, jobChan, resultChan, &wg, config)
	}
	
	// Send jobs
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Wait for completion and close results
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	var results []FeedProcessingResult
	for result := range resultChan {
		results = append(results, result)
	}
	
	return results
}

// createProcessingJobs creates jobs for all configured feeds
func (p *ParallelRSSProcessor) createProcessingJobs(priorities map[string]int) []FeedProcessingJob {
	var jobs []FeedProcessingJob
	
	for source, url := range CryptoRSSFeeds {
		priority := 1
		if priorities != nil {
			if p, exists := priorities[source]; exists {
				priority = p
			}
		}
		
		jobs = append(jobs, FeedProcessingJob{
			Source:   source,
			URL:      url,
			Priority: priority,
		})
	}
	
	p.sortJobsByPriority(jobs)
	return jobs
}

// sortJobsByPriority sorts jobs by priority (higher first)
func (p *ParallelRSSProcessor) sortJobsByPriority(jobs []FeedProcessingJob) {
	for i := 0; i < len(jobs)-1; i++ {
		for j := i + 1; j < len(jobs); j++ {
			if jobs[i].Priority < jobs[j].Priority {
				jobs[i], jobs[j] = jobs[j], jobs[i]
			}
		}
	}
}

// aggregateResults combines individual results into a batch summary
func (p *ParallelRSSProcessor) aggregateResults(results []FeedProcessingResult, totalTime time.Duration) BatchProcessingResult {
	summary := BatchProcessingResult{
		Results:   results,
		TotalTime: totalTime,
	}
	
	var totalSentiment float64
	var sentimentCount int
	
	for _, result := range results {
		summary.TotalItems += len(result.NewsItems)
		
		if result.Error != nil {
			summary.ErrorCount++
		} else {
			summary.SuccessCount++
		}
		
		if result.CacheHit {
			summary.CacheHitCount++
		}
		
		if result.SentimentData != nil {
			totalSentiment += result.SentimentData.OverallScore
			sentimentCount++
		}
	}
	
	if sentimentCount > 0 {
		summary.AverageSentiment = totalSentiment / float64(sentimentCount)
	}
	
	return summary
}

// GetProcessingStats returns statistics about processing performance
func (p *ParallelRSSProcessor) GetProcessingStats() map[string]interface{} {
	stats := map[string]interface{}{
		"max_workers": p.maxWorkers,
		"timeout":     p.timeout,
	}
	
	if p.cacheManager != nil {
		cacheStats := p.cacheManager.GetStats()
		stats["cache_stats"] = cacheStats
	}
	
	if p.healthMonitor != nil {
		healthSummary := p.healthMonitor.GetHealthSummary()
		stats["health_summary"] = healthSummary
	}
	
	return stats
}

// ProcessRecentNewsParallel processes recent news from multiple timeframes concurrently
func (p *ParallelRSSProcessor) ProcessRecentNewsParallel(ctx context.Context, timeframes []int, config ProcessingConfig) map[int]BatchProcessingResult {
	results := make(map[int]BatchProcessingResult)
	var wg sync.WaitGroup
	var mutex sync.Mutex
	
	for _, hours := range timeframes {
		wg.Add(1)
		go func(h int) {
			defer wg.Done()
			
			// Check cache first
			if config.UseCache && p.cacheManager != nil {
				if cachedNews, hit := p.cacheManager.GetCachedRecentNews(h); hit {
					mutex.Lock()
					results[h] = BatchProcessingResult{
						Results: []FeedProcessingResult{{
							NewsItems:  cachedNews,
							CacheHit:   true,
							ProcessingTime: 0,
						}},
						TotalItems:   len(cachedNews),
						SuccessCount: 1,
						CacheHitCount: 1,
					}
					mutex.Unlock()
					return
				}
			}
			
			// Fetch and process
			startTime := time.Now()
			recentNews, err := p.feedReader.FetchRecentNews(h)
			processingTime := time.Since(startTime)
			
			batchResult := BatchProcessingResult{
				TotalTime: processingTime,
			}
			
			if err != nil {
				batchResult.ErrorCount = 1
				batchResult.Results = []FeedProcessingResult{{
					Error: err,
					ProcessingTime: processingTime,
				}}
			} else {
				batchResult.TotalItems = len(recentNews)
				batchResult.SuccessCount = 1
				batchResult.Results = []FeedProcessingResult{{
					NewsItems: recentNews,
					ProcessingTime: processingTime,
				}}
				
				// Cache the results
				if config.UseCache && p.cacheManager != nil {
					p.cacheManager.CacheRecentNews(h, recentNews)
				}
			}
			
			mutex.Lock()
			results[h] = batchResult
			mutex.Unlock()
		}(hours)
	}
	
	wg.Wait()
	return results
}
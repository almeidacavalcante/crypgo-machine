package external

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

type RSSCacheManager struct {
	cache         map[string]*CacheEntry
	cacheMutex    sync.RWMutex
	defaultTTL    time.Duration
	maxCacheSize  int
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

type CacheEntry struct {
	Key           string                  `json:"key"`
	Data          []NewsItem             `json:"data"`
	SentimentData *NewsAnalysisResult    `json:"sentiment_data,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	AccessCount   int                    `json:"access_count"`
	LastAccessed  time.Time              `json:"last_accessed"`
	Source        string                 `json:"source"`
	ContentHash   string                 `json:"content_hash"`
}

type CacheStats struct {
	TotalEntries    int           `json:"total_entries"`
	HitCount        int64         `json:"hit_count"`
	MissCount       int64         `json:"miss_count"`
	HitRate         float64       `json:"hit_rate"`
	MemoryUsage     int64         `json:"memory_usage_bytes"`
	OldestEntry     time.Time     `json:"oldest_entry"`
	NewestEntry     time.Time     `json:"newest_entry"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

type CacheConfig struct {
	DefaultTTL      time.Duration
	MaxCacheSize    int
	CleanupInterval time.Duration
}

func NewRSSCacheManager(config CacheConfig) *RSSCacheManager {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 15 * time.Minute
	}
	if config.MaxCacheSize == 0 {
		config.MaxCacheSize = 1000
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	
	manager := &RSSCacheManager{
		cache:        make(map[string]*CacheEntry),
		defaultTTL:   config.DefaultTTL,
		maxCacheSize: config.MaxCacheSize,
		stopCleanup:  make(chan struct{}),
	}
	
	// Start cleanup routine
	manager.startCleanupRoutine(config.CleanupInterval)
	
	return manager
}

func (c *RSSCacheManager) startCleanupRoutine(interval time.Duration) {
	c.cleanupTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanup()
			case <-c.stopCleanup:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()
}

func (c *RSSCacheManager) Stop() {
	close(c.stopCleanup)
}

// GetCachedFeed retrieves cached news items for a specific source
func (c *RSSCacheManager) GetCachedFeed(source string) ([]NewsItem, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	key := c.generateFeedKey(source)
	entry, exists := c.cache[key]
	
	if !exists {
		return nil, false
	}
	
	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	// Update access statistics
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	
	return entry.Data, true
}

// CacheFeed stores news items in cache with intelligent TTL based on content freshness
func (c *RSSCacheManager) CacheFeed(source string, newsItems []NewsItem) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	if len(newsItems) == 0 {
		return
	}
	
	key := c.generateFeedKey(source)
	contentHash := c.generateContentHash(newsItems)
	
	// Check if content has changed
	if existingEntry, exists := c.cache[key]; exists {
		if existingEntry.ContentHash == contentHash {
			// Content hasn't changed, just update expiration
			existingEntry.ExpiresAt = time.Now().Add(c.defaultTTL)
			return
		}
	}
	
	// Calculate TTL based on content freshness
	ttl := c.calculateTTL(source, newsItems)
	
	entry := &CacheEntry{
		Key:          key,
		Data:         newsItems,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ttl),
		AccessCount:  0,
		LastAccessed: time.Now(),
		Source:       source,
		ContentHash:  contentHash,
	}
	
	c.cache[key] = entry
	
	// Enforce cache size limit
	c.enforceMaxSize()
}

// GetCachedSentiment retrieves cached sentiment analysis for a source
func (c *RSSCacheManager) GetCachedSentiment(source string) (*NewsAnalysisResult, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	key := c.generateSentimentKey(source)
	entry, exists := c.cache[key]
	
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	
	return entry.SentimentData, true
}

// CacheSentiment stores sentiment analysis results
func (c *RSSCacheManager) CacheSentiment(source string, sentiment *NewsAnalysisResult) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	key := c.generateSentimentKey(source)
	
	entry := &CacheEntry{
		Key:           key,
		SentimentData: sentiment,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(c.defaultTTL),
		AccessCount:   0,
		LastAccessed:  time.Now(),
		Source:        source,
	}
	
	c.cache[key] = entry
	c.enforceMaxSize()
}

// GetCachedRecentNews retrieves cached recent news with time filtering
func (c *RSSCacheManager) GetCachedRecentNews(hoursBack int) ([]NewsItem, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	key := c.generateRecentNewsKey(hoursBack)
	entry, exists := c.cache[key]
	
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	entry.AccessCount++
	entry.LastAccessed = time.Now()
	
	return entry.Data, true
}

// CacheRecentNews stores filtered recent news
func (c *RSSCacheManager) CacheRecentNews(hoursBack int, newsItems []NewsItem) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	key := c.generateRecentNewsKey(hoursBack)
	
	// Shorter TTL for time-filtered content
	ttl := time.Duration(hoursBack/2) * time.Hour
	if ttl < 5*time.Minute {
		ttl = 5 * time.Minute
	}
	
	entry := &CacheEntry{
		Key:          key,
		Data:         newsItems,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ttl),
		AccessCount:  0,
		LastAccessed: time.Now(),
		Source:       fmt.Sprintf("recent_%dh", hoursBack),
	}
	
	c.cache[key] = entry
	c.enforceMaxSize()
}

// InvalidateSource removes all cache entries for a specific source
func (c *RSSCacheManager) InvalidateSource(source string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	keysToDelete := []string{
		c.generateFeedKey(source),
		c.generateSentimentKey(source),
	}
	
	for _, key := range keysToDelete {
		delete(c.cache, key)
	}
}

// InvalidateAll clears the entire cache
func (c *RSSCacheManager) InvalidateAll() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	c.cache = make(map[string]*CacheEntry)
}

// GetStats returns cache statistics
func (c *RSSCacheManager) GetStats() CacheStats {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	var hitCount, missCount int64
	var oldestEntry, newestEntry time.Time
	var memoryUsage int64
	
	first := true
	for _, entry := range c.cache {
		if first {
			oldestEntry = entry.CreatedAt
			newestEntry = entry.CreatedAt
			first = false
		} else {
			if entry.CreatedAt.Before(oldestEntry) {
				oldestEntry = entry.CreatedAt
			}
			if entry.CreatedAt.After(newestEntry) {
				newestEntry = entry.CreatedAt
			}
		}
		
		// Rough memory usage calculation
		entrySize := len(entry.Key) + len(entry.Source) + len(entry.ContentHash)
		for _, item := range entry.Data {
			entrySize += len(item.Title) + len(item.Description) + len(item.Link) + len(item.Content)
		}
		memoryUsage += int64(entrySize)
	}
	
	// Note: Hit/miss counts would need to be tracked separately for accuracy
	// This is a simplified implementation
	totalAccess := hitCount + missCount
	hitRate := 0.0
	if totalAccess > 0 {
		hitRate = float64(hitCount) / float64(totalAccess)
	}
	
	return CacheStats{
		TotalEntries:    len(c.cache),
		HitCount:        hitCount,
		MissCount:       missCount,
		HitRate:         hitRate,
		MemoryUsage:     memoryUsage,
		OldestEntry:     oldestEntry,
		NewestEntry:     newestEntry,
		CleanupInterval: 5 * time.Minute, // Fixed interval
	}
}

// Private methods

func (c *RSSCacheManager) generateFeedKey(source string) string {
	return fmt.Sprintf("feed:%s", source)
}

func (c *RSSCacheManager) generateSentimentKey(source string) string {
	return fmt.Sprintf("sentiment:%s", source)
}

func (c *RSSCacheManager) generateRecentNewsKey(hoursBack int) string {
	return fmt.Sprintf("recent:%dh", hoursBack)
}

func (c *RSSCacheManager) generateContentHash(newsItems []NewsItem) string {
	// Create a hash based on titles and publication times
	var content string
	for _, item := range newsItems {
		content += item.Title + item.PublishedAt.Format(time.RFC3339)
	}
	
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func (c *RSSCacheManager) calculateTTL(source string, newsItems []NewsItem) time.Duration {
	if len(newsItems) == 0 {
		return c.defaultTTL
	}
	
	// Find the newest article
	var newestTime time.Time
	for _, item := range newsItems {
		if item.PublishedAt.After(newestTime) {
			newestTime = item.PublishedAt
		}
	}
	
	// Calculate TTL based on content freshness
	age := time.Since(newestTime)
	
	switch {
	case age < 30*time.Minute:
		// Very fresh content - cache for shorter time
		return 5 * time.Minute
	case age < 2*time.Hour:
		// Recent content - normal cache time
		return 10 * time.Minute
	case age < 24*time.Hour:
		// Older content - cache longer
		return 30 * time.Minute
	default:
		// Very old content - cache even longer
		return 1 * time.Hour
	}
}

func (c *RSSCacheManager) cleanup() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	
	now := time.Now()
	keysToDelete := []string{}
	
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(c.cache, key)
	}
}

func (c *RSSCacheManager) enforceMaxSize() {
	if len(c.cache) <= c.maxCacheSize {
		return
	}
	
	// Remove oldest entries based on last access time
	type entryInfo struct {
		key          string
		lastAccessed time.Time
	}
	
	var entries []entryInfo
	for key, entry := range c.cache {
		entries = append(entries, entryInfo{
			key:          key,
			lastAccessed: entry.LastAccessed,
		})
	}
	
	// Sort by last accessed time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].lastAccessed.After(entries[j].lastAccessed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Remove oldest entries until we're under the limit
	entriesToRemove := len(c.cache) - c.maxCacheSize
	for i := 0; i < entriesToRemove; i++ {
		delete(c.cache, entries[i].key)
	}
}

// IsContentFresh checks if cached content is still fresh enough to use
func (c *RSSCacheManager) IsContentFresh(source string, maxAge time.Duration) bool {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	
	key := c.generateFeedKey(source)
	entry, exists := c.cache[key]
	
	if !exists {
		return false
	}
	
	return time.Since(entry.CreatedAt) < maxAge
}

// WarmCache pre-loads cache with provided data
func (c *RSSCacheManager) WarmCache(sourceData map[string][]NewsItem) {
	for source, newsItems := range sourceData {
		c.CacheFeed(source, newsItems)
	}
}
package external

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type RSSHealthMonitor struct {
	httpClient    *http.Client
	feedStatuses  map[string]*FeedStatus
	statusMutex   sync.RWMutex
	checkInterval time.Duration
	stopChan      chan struct{}
	running       bool
}

type FeedStatus struct {
	Source           string        `json:"source"`
	URL              string        `json:"url"`
	IsHealthy        bool          `json:"is_healthy"`
	LastCheckTime    time.Time     `json:"last_check_time"`
	LastSuccessTime  time.Time     `json:"last_success_time"`
	LastErrorTime    time.Time     `json:"last_error_time"`
	LastError        string        `json:"last_error,omitempty"`
	ResponseTime     time.Duration `json:"response_time"`
	StatusCode       int           `json:"status_code"`
	ContentLength    int64         `json:"content_length"`
	ItemCount        int           `json:"item_count"`
	ConsecutiveErrors int          `json:"consecutive_errors"`
	SuccessRate      float64       `json:"success_rate"`
	TotalChecks      int           `json:"total_checks"`
	SuccessfulChecks int           `json:"successful_checks"`
}

type HealthSummary struct {
	TotalFeeds      int           `json:"total_feeds"`
	HealthyFeeds    int           `json:"healthy_feeds"`
	UnhealthyFeeds  int           `json:"unhealthy_feeds"`
	OverallHealth   float64       `json:"overall_health"`
	LastCheckTime   time.Time     `json:"last_check_time"`
	FeedStatuses    []FeedStatus  `json:"feed_statuses"`
	Recommendations []string      `json:"recommendations,omitempty"`
}

type FeedHealthCheck struct {
	Source      string
	URL         string
	IsHealthy   bool
	Error       error
	ResponseTime time.Duration
	StatusCode  int
	ItemCount   int
}

func NewRSSHealthMonitor() *RSSHealthMonitor {
	monitor := &RSSHealthMonitor{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		feedStatuses:  make(map[string]*FeedStatus),
		checkInterval: 5 * time.Minute, // Check every 5 minutes
		stopChan:      make(chan struct{}),
		running:       false,
	}
	
	// Initialize feed statuses
	for source, url := range CryptoRSSFeeds {
		monitor.feedStatuses[source] = &FeedStatus{
			Source:      source,
			URL:         url,
			IsHealthy:   true, // Assume healthy until proven otherwise
			LastCheckTime: time.Now(),
		}
	}
	
	return monitor
}

func (h *RSSHealthMonitor) StartMonitoring() {
	h.statusMutex.Lock()
	if h.running {
		h.statusMutex.Unlock()
		return
	}
	h.running = true
	h.statusMutex.Unlock()
	
	// Initial health check
	h.CheckAllFeeds()
	
	// Start periodic monitoring
	go func() {
		ticker := time.NewTicker(h.checkInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				h.CheckAllFeeds()
			case <-h.stopChan:
				return
			}
		}
	}()
}

func (h *RSSHealthMonitor) StopMonitoring() {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()
	
	if h.running {
		close(h.stopChan)
		h.running = false
	}
}

func (h *RSSHealthMonitor) CheckAllFeeds() []FeedHealthCheck {
	var checks []FeedHealthCheck
	var wg sync.WaitGroup
	checksChan := make(chan FeedHealthCheck, len(CryptoRSSFeeds))
	
	// Check all feeds concurrently
	for source, url := range CryptoRSSFeeds {
		wg.Add(1)
		go func(src, feedURL string) {
			defer wg.Done()
			check := h.checkSingleFeed(src, feedURL)
			checksChan <- check
		}(source, url)
	}
	
	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(checksChan)
	}()
	
	// Collect results and update statuses
	for check := range checksChan {
		checks = append(checks, check)
		h.updateFeedStatus(check)
	}
	
	return checks
}

func (h *RSSHealthMonitor) checkSingleFeed(source, url string) FeedHealthCheck {
	startTime := time.Now()
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FeedHealthCheck{
			Source:       source,
			URL:          url,
			IsHealthy:    false,
			Error:        fmt.Errorf("failed to create request: %w", err),
			ResponseTime: time.Since(startTime),
		}
	}
	
	req.Header.Set("User-Agent", "CrypGo-HealthMonitor/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return FeedHealthCheck{
			Source:       source,
			URL:          url,
			IsHealthy:    false,
			Error:        fmt.Errorf("request failed: %w", err),
			ResponseTime: time.Since(startTime),
		}
	}
	defer resp.Body.Close()
	
	responseTime := time.Since(startTime)
	
	if resp.StatusCode != http.StatusOK {
		return FeedHealthCheck{
			Source:       source,
			URL:          url,
			IsHealthy:    false,
			Error:        fmt.Errorf("HTTP %d", resp.StatusCode),
			ResponseTime: responseTime,
			StatusCode:   resp.StatusCode,
		}
	}
	
	// Try to parse the RSS to ensure it's valid
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FeedHealthCheck{
			Source:       source,
			URL:          url,
			IsHealthy:    false,
			Error:        fmt.Errorf("failed to read response: %w", err),
			ResponseTime: responseTime,
			StatusCode:   resp.StatusCode,
		}
	}
	
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return FeedHealthCheck{
			Source:       source,
			URL:          url,
			IsHealthy:    false,
			Error:        fmt.Errorf("invalid RSS format: %w", err),
			ResponseTime: responseTime,
			StatusCode:   resp.StatusCode,
		}
	}
	
	// Successful check
	return FeedHealthCheck{
		Source:       source,
		URL:          url,
		IsHealthy:    true,
		Error:        nil,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		ItemCount:    len(feed.Channel.Items),
	}
}

func (h *RSSHealthMonitor) updateFeedStatus(check FeedHealthCheck) {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()
	
	status, exists := h.feedStatuses[check.Source]
	if !exists {
		status = &FeedStatus{
			Source: check.Source,
			URL:    check.URL,
		}
		h.feedStatuses[check.Source] = status
	}
	
	now := time.Now()
	status.LastCheckTime = now
	status.ResponseTime = check.ResponseTime
	status.StatusCode = check.StatusCode
	status.ItemCount = check.ItemCount
	status.TotalChecks++
	
	if check.IsHealthy {
		status.IsHealthy = true
		status.LastSuccessTime = now
		status.ConsecutiveErrors = 0
		status.SuccessfulChecks++
		status.LastError = ""
	} else {
		status.IsHealthy = false
		status.LastErrorTime = now
		status.ConsecutiveErrors++
		if check.Error != nil {
			status.LastError = check.Error.Error()
		}
	}
	
	// Calculate success rate
	if status.TotalChecks > 0 {
		status.SuccessRate = float64(status.SuccessfulChecks) / float64(status.TotalChecks)
	}
}

func (h *RSSHealthMonitor) GetHealthSummary() HealthSummary {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	
	var statuses []FeedStatus
	healthyCount := 0
	totalFeeds := len(h.feedStatuses)
	
	for _, status := range h.feedStatuses {
		statuses = append(statuses, *status)
		if status.IsHealthy {
			healthyCount++
		}
	}
	
	overallHealth := 0.0
	if totalFeeds > 0 {
		overallHealth = float64(healthyCount) / float64(totalFeeds)
	}
	
	summary := HealthSummary{
		TotalFeeds:     totalFeeds,
		HealthyFeeds:   healthyCount,
		UnhealthyFeeds: totalFeeds - healthyCount,
		OverallHealth:  overallHealth,
		LastCheckTime:  time.Now(),
		FeedStatuses:   statuses,
	}
	
	// Generate recommendations
	summary.Recommendations = h.generateRecommendations(statuses, overallHealth)
	
	return summary
}

func (h *RSSHealthMonitor) generateRecommendations(statuses []FeedStatus, overallHealth float64) []string {
	var recommendations []string
	
	// Overall health recommendations
	switch {
	case overallHealth < 0.5:
		recommendations = append(recommendations, "Critical: More than half of RSS feeds are unhealthy. Consider implementing fallback data sources.")
	case overallHealth < 0.8:
		recommendations = append(recommendations, "Warning: Some RSS feeds are experiencing issues. Monitor closely.")
	default:
		recommendations = append(recommendations, "Good: Most RSS feeds are healthy.")
	}
	
	// Individual feed recommendations
	for _, status := range statuses {
		if !status.IsHealthy {
			if status.ConsecutiveErrors > 10 {
				recommendations = append(recommendations, 
					fmt.Sprintf("Critical: %s has failed %d consecutive times. Consider removing or replacing this source.", 
						status.Source, status.ConsecutiveErrors))
			} else if status.ConsecutiveErrors > 5 {
				recommendations = append(recommendations, 
					fmt.Sprintf("Warning: %s is experiencing frequent failures (%d consecutive errors).", 
						status.Source, status.ConsecutiveErrors))
			}
		}
		
		if status.ResponseTime > 10*time.Second {
			recommendations = append(recommendations, 
				fmt.Sprintf("Performance: %s has slow response times (%.2fs). Consider timeout adjustments.", 
					status.Source, status.ResponseTime.Seconds()))
		}
		
		if status.SuccessRate < 0.9 && status.TotalChecks > 10 {
			recommendations = append(recommendations, 
				fmt.Sprintf("Reliability: %s has low success rate (%.1f%%). Investigate source reliability.", 
					status.Source, status.SuccessRate*100))
		}
	}
	
	return recommendations
}

func (h *RSSHealthMonitor) GetFeedStatus(source string) (*FeedStatus, bool) {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	
	status, exists := h.feedStatuses[source]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid race conditions
	statusCopy := *status
	return &statusCopy, true
}

func (h *RSSHealthMonitor) IsRunning() bool {
	h.statusMutex.RLock()
	defer h.statusMutex.RUnlock()
	return h.running
}

func (h *RSSHealthMonitor) SetCheckInterval(interval time.Duration) {
	h.statusMutex.Lock()
	defer h.statusMutex.Unlock()
	h.checkInterval = interval
}

// ValidateAllFeeds performs an immediate health check and returns any issues
func (h *RSSHealthMonitor) ValidateAllFeeds() error {
	checks := h.CheckAllFeeds()
	
	var failedFeeds []string
	for _, check := range checks {
		if !check.IsHealthy {
			failedFeeds = append(failedFeeds, fmt.Sprintf("%s: %v", check.Source, check.Error))
		}
	}
	
	if len(failedFeeds) == len(checks) {
		return fmt.Errorf("all RSS feeds failed validation: %v", failedFeeds)
	}
	
	if len(failedFeeds) > 0 {
		return fmt.Errorf("some RSS feeds failed validation: %v", failedFeeds)
	}
	
	return nil
}
package scheduler

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type SentimentScheduler struct {
	marketService *service.MarketSentimentService
	repository    repository.SentimentSuggestionRepository
	messageBroker queue.MessageBroker

	// mu guards starting, running and stopChannel: Start runs in its own goroutine
	// while HTTP handlers call IsRunning, Stop and TriggerManualAnalysis.
	mu          sync.Mutex
	stopChannel chan bool
	starting    bool
	running     bool
}

type SchedulerConfig struct {
	AnalysisInterval   time.Duration // How often to run full analysis
	QuickCheckInterval time.Duration // How often to run quick checks
	MaxRetries         int           // Max retries on failure
	NotifyOnSuggestion bool          // Send notifications when suggestions are generated
}

type SentimentNotificationPayload struct {
	SuggestionID string                              `json:"suggestion_id"`
	Sentiment    string                              `json:"sentiment"`
	Score        float64                             `json:"score"`
	Confidence   float64                             `json:"confidence"`
	Reasoning    string                              `json:"reasoning"`
	Suggestions  service.SentimentTradingSuggestions `json:"suggestions"`
	Timestamp    time.Time                           `json:"timestamp"`
	Type         string                              `json:"type"` // "full_analysis" or "quick_check"
}

func NewSentimentScheduler(
	marketService *service.MarketSentimentService,
	repository repository.SentimentSuggestionRepository,
	messageBroker queue.MessageBroker,
) *SentimentScheduler {
	return &SentimentScheduler{
		marketService: marketService,
		repository:    repository,
		messageBroker: messageBroker,
		stopChannel:   make(chan bool),
		running:       false,
	}
}

// Start begins the scheduled sentiment analysis
func (s *SentimentScheduler) Start(config SchedulerConfig) error {
	if err := s.beginStart(); err != nil {
		return err
	}

	log.Println("🕐 Starting Sentiment Analysis Scheduler")
	log.Printf("📅 Full Analysis Interval: %v", config.AnalysisInterval)
	log.Printf("⚡ Quick Check Interval: %v", config.QuickCheckInterval)

	// Validate data sources before starting. This talks to the network, so it
	// runs without holding the lock; the scheduler only reports "running" once
	// validation has passed.
	if err := s.marketService.ValidateDataSources(); err != nil {
		s.finishStart(false)
		return fmt.Errorf("failed to validate data sources: %w", err)
	}

	stop := s.finishStart(true)

	// Start the scheduler goroutines
	go s.runFullAnalysisScheduler(config, stop)
	go s.runQuickCheckScheduler(config, stop)

	log.Println("✅ Sentiment Analysis Scheduler started successfully")
	return nil
}

// beginStart reserves the scheduler for a single Start call.
func (s *SentimentScheduler) beginStart() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("sentiment scheduler is already running")
	}
	if s.starting {
		return fmt.Errorf("sentiment scheduler is already starting")
	}
	s.starting = true
	return nil
}

// finishStart ends the start attempt begun by beginStart. When started is true
// the scheduler becomes running and the returned channel is closed by Stop;
// otherwise it stays stopped and nil is returned.
func (s *SentimentScheduler) finishStart(started bool) <-chan bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.starting = false
	if !started {
		return nil
	}

	// A fresh channel per run, so the scheduler can be started again after Stop.
	s.stopChannel = make(chan bool)
	s.running = true
	return s.stopChannel
}

// Stop halts the sentiment analysis scheduler
func (s *SentimentScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	log.Println("⏹️ Stopping Sentiment Analysis Scheduler")
	s.running = false
	close(s.stopChannel)
	log.Println("✅ Sentiment Analysis Scheduler stopped")
}

// runFullAnalysisScheduler runs comprehensive sentiment analysis at regular intervals
func (s *SentimentScheduler) runFullAnalysisScheduler(config SchedulerConfig, stop <-chan bool) {
	ticker := time.NewTicker(config.AnalysisInterval)
	defer ticker.Stop()

	// Run initial analysis immediately
	s.performFullAnalysis(config)

	for {
		select {
		case <-ticker.C:
			s.performFullAnalysis(config)
		case <-stop:
			log.Println("📊 Full analysis scheduler stopped")
			return
		}
	}
}

// runQuickCheckScheduler runs lightweight sentiment checks more frequently
func (s *SentimentScheduler) runQuickCheckScheduler(config SchedulerConfig, stop <-chan bool) {
	ticker := time.NewTicker(config.QuickCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performQuickCheck(config)
		case <-stop:
			log.Println("⚡ Quick check scheduler stopped")
			return
		}
	}
}

// performFullAnalysis executes comprehensive sentiment analysis
func (s *SentimentScheduler) performFullAnalysis(config SchedulerConfig) {
	log.Println("📊 Starting scheduled full sentiment analysis...")

	var lastErr error
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		result, err := s.marketService.CollectMarketSentiment()
		if err != nil {
			lastErr = err
			log.Printf("❌ Full analysis attempt %d/%d failed: %v", attempt, config.MaxRetries, err)
			if attempt < config.MaxRetries {
				time.Sleep(time.Duration(attempt) * time.Minute) // Exponential backoff
				continue
			}
			break
		}

		// Save suggestion to repository
		if err := s.repository.Save(result.Suggestion); err != nil {
			log.Printf("⚠️ Failed to save sentiment suggestion: %v", err)
			// Continue anyway - don't fail the whole analysis
		}

		// Send notification if configured
		if config.NotifyOnSuggestion {
			s.sendNotification(result, "full_analysis")
		}

		log.Printf("✅ Full sentiment analysis completed - Sentiment: %s, Score: %.3f, Confidence: %.2f",
			result.Suggestion.GetLevel(), result.Suggestion.GetOverallScore(), result.Confidence)
		return
	}

	log.Printf("❌ Full sentiment analysis failed after %d attempts: %v", config.MaxRetries, lastErr)
}

// performQuickCheck executes lightweight sentiment monitoring
func (s *SentimentScheduler) performQuickCheck(config SchedulerConfig) {
	result, err := s.marketService.QuickSentimentCheck()
	if err != nil {
		log.Printf("⚠️ Quick sentiment check failed: %v", err)
		return
	}

	log.Printf("⚡ Quick check - Sentiment: %s, Score: %.3f",
		result.Suggestion.GetLevel(), result.Suggestion.GetOverallScore())

	// Only notify on significant sentiment changes or extreme values
	if s.shouldNotifyQuickCheck(result) && config.NotifyOnSuggestion {
		s.sendNotification(result, "quick_check")
	}
}

// shouldNotifyQuickCheck determines if a quick check warrants notification
func (s *SentimentScheduler) shouldNotifyQuickCheck(result *service.SentimentCollectionResult) bool {
	sentiment := result.Suggestion.GetLevel()
	score := result.Suggestion.GetOverallScore()

	// Notify on extreme sentiment levels
	return sentiment.String() == "very_bullish" || sentiment.String() == "very_bearish" ||
		score.GetValue() > 0.5 || score.GetValue() < -0.5
}

// sendNotification sends sentiment analysis results via message broker
func (s *SentimentScheduler) sendNotification(result *service.SentimentCollectionResult, analysisType string) {
	sentiment := result.Suggestion.GetLevel()
	suggestions := s.marketService.GetSentimentSuggestions(sentiment.String())

	payload := SentimentNotificationPayload{
		SuggestionID: result.Suggestion.GetId().GetValue(),
		Sentiment:    sentiment.String(),
		Score:        result.Suggestion.GetOverallScore().GetValue(),
		Confidence:   result.Confidence,
		Reasoning:    result.Reasoning,
		Suggestions:  suggestions,
		Timestamp:    time.Now(),
		Type:         analysisType,
	}

	messageBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Failed to marshal sentiment notification: %v", err)
		return
	}

	message := queue.Message{
		RoutingKey: "sentiment.analysis.completed",
		Payload:    messageBytes,
	}

	if err := s.messageBroker.Publish("trading_bot", message); err != nil {
		log.Printf("❌ Failed to publish sentiment notification: %v", err)
	} else {
		log.Printf("📨 Sentiment notification sent - %s analysis, sentiment: %s", analysisType, sentiment.String())
	}
}

// GetDefaultConfig returns recommended scheduler configuration
func GetDefaultConfig() SchedulerConfig {
	return SchedulerConfig{
		AnalysisInterval:   4 * time.Hour, // Full analysis every 4 hours as per plan
		QuickCheckInterval: 1 * time.Hour, // Quick checks every hour
		MaxRetries:         3,             // Retry up to 3 times on failure
		NotifyOnSuggestion: true,          // Send notifications
	}
}

// GetDevelopmentConfig returns configuration suitable for development/testing
func GetDevelopmentConfig() SchedulerConfig {
	return SchedulerConfig{
		AnalysisInterval:   30 * time.Minute, // More frequent for testing
		QuickCheckInterval: 10 * time.Minute, // More frequent for testing
		MaxRetries:         2,
		NotifyOnSuggestion: true,
	}
}

// IsRunning returns whether the scheduler is currently running
func (s *SentimentScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// TriggerManualAnalysis forces an immediate full sentiment analysis
func (s *SentimentScheduler) TriggerManualAnalysis() error {
	if !s.IsRunning() {
		return fmt.Errorf("scheduler is not running")
	}

	log.Println("🔄 Manual sentiment analysis triggered")
	config := GetDefaultConfig()
	go s.performFullAnalysis(config)

	return nil
}

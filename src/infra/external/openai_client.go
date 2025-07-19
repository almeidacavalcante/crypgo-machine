package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OpenAIClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
	rateLimiter *RateLimiter
	stats       *OpenAIStats
	statsMutex  sync.RWMutex
}

type OpenAIConfig struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	MaxRequests int
	TimeWindow  time.Duration
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type OpenAIStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalTokensUsed    int64         `json:"total_tokens_used"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastRequestTime    time.Time     `json:"last_request_time"`
	EstimatedCost      float64       `json:"estimated_cost_usd"`
}

type RateLimiter struct {
	requests    []time.Time
	maxRequests int
	timeWindow  time.Duration
	mutex       sync.Mutex
}

type LLMAnalysisResult struct {
	Summary   string  `json:"summary"`
	Score     float64 `json:"score"`
	Reasoning string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

func NewOpenAIClient(config OpenAIConfig) *OpenAIClient {
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if config.Model == "" {
		config.Model = getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini")
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = getEnvIntOrDefault("OPENAI_MAX_TOKENS", 300)
	}
	if config.Temperature == 0.0 {
		if tempStr := os.Getenv("OPENAI_TEMPERATURE"); tempStr != "" {
			if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
				config.Temperature = temp
			}
		} else {
			config.Temperature = 0.2
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 60 // OpenAI tier 1 limit
	}
	if config.TimeWindow == 0 {
		config.TimeWindow = time.Minute
	}

	return &OpenAIClient{
		apiKey:      config.APIKey,
		model:       config.Model,
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		rateLimiter: &RateLimiter{
			maxRequests: config.MaxRequests,
			timeWindow:  config.TimeWindow,
		},
		stats: &OpenAIStats{},
	}
}

func (c *OpenAIClient) AnalyzeSentiment(ctx context.Context, content string) (*LLMAnalysisResult, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Check rate limiting
	if !c.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	}

	startTime := time.Now()
	c.updateStats(func(stats *OpenAIStats) {
		stats.TotalRequests++
		stats.LastRequestTime = startTime
	})

	prompt := c.buildSentimentPrompt(content)
	
	request := OpenAIRequest{
		Model:       c.model,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a cryptocurrency market sentiment analyst with deep expertise in crypto markets, DeFi, regulations, and trading psychology.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	response, err := c.makeRequest(ctx, request)
	if err != nil {
		c.updateStats(func(stats *OpenAIStats) {
			stats.FailedRequests++
		})
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	if response.Error != nil {
		c.updateStats(func(stats *OpenAIStats) {
			stats.FailedRequests++
		})
		return nil, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		c.updateStats(func(stats *OpenAIStats) {
			stats.FailedRequests++
		})
		return nil, fmt.Errorf("no response choices received from OpenAI")
	}

	// Update successful stats
	responseTime := time.Since(startTime)
	c.updateStats(func(stats *OpenAIStats) {
		stats.SuccessfulRequests++
		stats.TotalTokensUsed += int64(response.Usage.TotalTokens)
		stats.AverageResponseTime = (stats.AverageResponseTime + responseTime) / 2
		// Estimate cost: gpt-4o-mini is ~$0.00015 per 1K tokens
		tokenCost := float64(response.Usage.TotalTokens) * 0.00015 / 1000
		stats.EstimatedCost += tokenCost
	})

	result, err := c.parseAnalysisResult(response.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return result, nil
}

func (c *OpenAIClient) buildSentimentPrompt(content string) string {
	return fmt.Sprintf(`Analyze this cryptocurrency news article and provide a structured analysis:

ARTICLE CONTENT:
%s

ANALYSIS REQUIREMENTS:
1. Create a brief, informative summary (1-2 sentences, max 100 words)
2. Assign a sentiment score between -1.0 and +1.0 where:
   • +1.0 = Extremely bullish/positive for crypto market
   • +0.5 = Moderately positive/bullish
   • 0.0 = Neutral market impact
   • -0.5 = Moderately negative/bearish  
   • -1.0 = Extremely bearish/negative for crypto market

3. Provide reasoning for your score (1-2 sentences)

IMPORTANT FACTORS TO CONSIDER:
• Regulatory developments (positive/negative)
• Institutional adoption and investments
• Technical developments and innovations
• Market sentiment and psychology
• Trading volume and price movements
• Security incidents or exploits
• Macroeconomic factors affecting crypto

OUTPUT FORMAT (respond with exactly this structure):
Summary: [your summary here]
Score: [numerical score between -1.0 and +1.0]
Reasoning: [your reasoning here]
Confidence: [confidence level 0.0-1.0]`, content)
}

func (c *OpenAIClient) parseAnalysisResult(content string) (*LLMAnalysisResult, error) {
	result := &LLMAnalysisResult{}
	
	lines := strings.Split(strings.TrimSpace(content), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "Summary:") {
			result.Summary = strings.TrimSpace(strings.TrimPrefix(line, "Summary:"))
		} else if strings.HasPrefix(line, "Score:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Score:"))
			if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				// Ensure score is within bounds
				if score > 1.0 {
					score = 1.0
				} else if score < -1.0 {
					score = -1.0
				}
				result.Score = score
			}
		} else if strings.HasPrefix(line, "Reasoning:") {
			result.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "Reasoning:"))
		} else if strings.HasPrefix(line, "Confidence:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(line, "Confidence:"))
			if conf, err := strconv.ParseFloat(confStr, 64); err == nil {
				if conf > 1.0 {
					conf = 1.0
				} else if conf < 0.0 {
					conf = 0.0
				}
				result.Confidence = conf
			}
		}
	}
	
	// Validation
	if result.Summary == "" {
		return nil, fmt.Errorf("summary not found in response")
	}
	
	// Set default confidence if not provided
	if result.Confidence == 0.0 {
		result.Confidence = 0.8 // Default high confidence
	}
	
	return result, nil
}

func (c *OpenAIClient) makeRequest(ctx context.Context, request OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if response.Error != nil {
			return &response, nil // Return response with error for proper handling
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return &response, nil
}

func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	
	// Remove requests outside the time window
	cutoff := now.Add(-rl.timeWindow)
	filtered := make([]time.Time, 0, len(rl.requests))
	for _, reqTime := range rl.requests {
		if reqTime.After(cutoff) {
			filtered = append(filtered, reqTime)
		}
	}
	rl.requests = filtered

	// Check if we can make a new request
	if len(rl.requests) >= rl.maxRequests {
		return false
	}

	// Add current request
	rl.requests = append(rl.requests, now)
	return true
}

func (c *OpenAIClient) GetStats() OpenAIStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	
	// Return a copy to prevent race conditions
	return *c.stats
}

func (c *OpenAIClient) updateStats(fn func(*OpenAIStats)) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	fn(c.stats)
}

func (c *OpenAIClient) IsConfigured() bool {
	return c.apiKey != "" && c.apiKey != "your-openai-api-key-here"
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// BatchAnalyzeSentiment processes multiple articles in a single request for cost optimization
func (c *OpenAIClient) BatchAnalyzeSentiment(ctx context.Context, articles []NewsItem) ([]LLMAnalysisResult, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles provided")
	}

	// For now, process individually - can be optimized later with batching
	results := make([]LLMAnalysisResult, 0, len(articles))
	
	for _, article := range articles {
		result, err := c.AnalyzeSentiment(ctx, article.Content)
		if err != nil {
			// Log error but continue with other articles
			continue
		}
		results = append(results, *result)
	}
	
	if len(results) == 0 {
		return nil, fmt.Errorf("failed to analyze any articles")
	}
	
	return results, nil
}
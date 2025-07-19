package external

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type FearGreedClient struct {
	httpClient *http.Client
	baseURL    string
}

type FearGreedIndexResponse struct {
	Name string `json:"name"`
	Data []struct {
		Value             string    `json:"value"`
		ValueClassification string `json:"value_classification"`
		Timestamp         string    `json:"timestamp"`
		TimeUntilUpdate   string    `json:"time_until_update"`
	} `json:"data"`
}

type FearGreedData struct {
	Value         int
	Classification string
	Timestamp     time.Time
}

func NewFearGreedClient() *FearGreedClient {
	return &FearGreedClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.alternative.me",
	}
}

func (f *FearGreedClient) GetLatestIndex() (*FearGreedData, error) {
	url := fmt.Sprintf("%s/fng/", f.baseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add User-Agent header as recommended in the plan
	req.Header.Set("User-Agent", "CrypGo-Sentiment-Bot/1.0")
	req.Header.Set("Accept", "application/json")
	
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var fearGreedResp FearGreedIndexResponse
	if err := json.Unmarshal(body, &fearGreedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(fearGreedResp.Data) == 0 {
		return nil, fmt.Errorf("no data received from Fear & Greed API")
	}
	
	// Parse the latest entry
	latest := fearGreedResp.Data[0]
	
	value, err := strconv.Atoi(latest.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fear greed value: %w", err)
	}
	
	// Parse timestamp (Unix timestamp)
	timestamp, err := strconv.ParseInt(latest.Timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	
	return &FearGreedData{
		Value:         value,
		Classification: latest.ValueClassification,
		Timestamp:     time.Unix(timestamp, 0),
	}, nil
}

// GetNormalizedScore converts Fear & Greed Index (0-100) to normalized score (-1 to +1)
// As described in the plan: (fearGreed - 50) / 50
func (f *FearGreedData) GetNormalizedScore() float64 {
	return float64(f.Value-50) / 50.0
}

// IsExtremeFear returns true if index indicates extreme fear (< 25)
func (f *FearGreedData) IsExtremeFear() bool {
	return f.Value < 25
}

// IsExtremeGreed returns true if index indicates extreme greed (> 75)
func (f *FearGreedData) IsExtremeGreed() bool {
	return f.Value > 75
}

// GetSentimentLevel returns human-readable sentiment level
func (f *FearGreedData) GetSentimentLevel() string {
	switch {
	case f.Value <= 20:
		return "Extreme Fear"
	case f.Value <= 40:
		return "Fear"
	case f.Value <= 60:
		return "Neutral"
	case f.Value <= 80:
		return "Greed"
	default:
		return "Extreme Greed"
	}
}
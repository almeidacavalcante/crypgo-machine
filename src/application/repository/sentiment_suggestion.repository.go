package repository

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
)

type SentimentSuggestionRepository interface {
	// Save a sentiment suggestion
	Save(suggestion *entity.SentimentSuggestion) error
	
	// Find suggestion by ID
	FindById(id *vo.EntityId) (*entity.SentimentSuggestion, error)
	
	// Find all pending suggestions
	FindPending() ([]*entity.SentimentSuggestion, error)
	
	// Find suggestions by status
	FindByStatus(status entity.SuggestionStatus) ([]*entity.SentimentSuggestion, error)
	
	// Find recent suggestions (limit)
	FindRecent(limit int) ([]*entity.SentimentSuggestion, error)
	
	// Find suggestions in date range
	FindByDateRange(from, to string) ([]*entity.SentimentSuggestion, error)
	
	// Update suggestion status and applied values
	Update(suggestion *entity.SentimentSuggestion) error
	
	// Get analytics: suggestion effectiveness, approval rates, etc.
	GetAnalytics() (*SentimentAnalytics, error)
	
	// Delete old suggestions (cleanup)
	DeleteOlderThan(days int) error
}

type SentimentAnalytics struct {
	TotalSuggestions     int     `json:"total_suggestions"`
	PendingSuggestions   int     `json:"pending_suggestions"`
	ApprovedSuggestions  int     `json:"approved_suggestions"`
	IgnoredSuggestions   int     `json:"ignored_suggestions"`
	CustomizedSuggestions int    `json:"customized_suggestions"`
	ApprovalRate         float64 `json:"approval_rate"`
	CustomizationRate    float64 `json:"customization_rate"`
	AvgResponseTimeHours float64 `json:"avg_response_time_hours"`
	LastSuggestionDate   string  `json:"last_suggestion_date"`
}
package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type SuggestionStatus string

const (
	StatusPending   SuggestionStatus = "pending"
	StatusApproved  SuggestionStatus = "approved"
	StatusIgnored   SuggestionStatus = "ignored"
	StatusCustomized SuggestionStatus = "customized"
)

type SentimentSuggestion struct {
	id                   *vo.EntityId
	overallScore         *vo.SentimentScore
	level                vo.SentimentLevel
	sources              *vo.SentimentSources
	suggestedMultiplier  float64
	suggestedThreshold   float64
	suggestedInterval    int
	reasoning            string
	confidence           float64
	status               SuggestionStatus
	userNotes            string
	appliedMultiplier    *float64
	appliedThreshold     *float64
	appliedInterval      *int
	createdAt            time.Time
	respondedAt          *time.Time
}

type SentimentSuggestionDTO struct {
	Id                   string                 `json:"id"`
	OverallScore         float64               `json:"overall_score"`
	Level                string                `json:"level"`
	LevelDisplay         string                `json:"level_display"`
	Sources              SentimentSourcesDTO   `json:"sources"`
	SuggestedMultiplier  float64               `json:"suggested_multiplier"`
	SuggestedThreshold   float64               `json:"suggested_threshold"`
	SuggestedInterval    int                   `json:"suggested_interval"`
	Reasoning            string                `json:"reasoning"`
	Confidence           float64               `json:"confidence"`
	Status               string                `json:"status"`
	UserNotes            string                `json:"user_notes,omitempty"`
	AppliedMultiplier    *float64              `json:"applied_multiplier,omitempty"`
	AppliedThreshold     *float64              `json:"applied_threshold,omitempty"`
	AppliedInterval      *int                  `json:"applied_interval,omitempty"`
	CreatedAt            time.Time             `json:"created_at"`
	RespondedAt          *time.Time            `json:"responded_at,omitempty"`
	ApprovalRequired     bool                  `json:"approval_required"`
}

type SentimentSourcesDTO struct {
	FearGreedIndex        int     `json:"fear_greed_index"`
	FearGreedClassification string `json:"fear_greed_classification"`
	NewsScore            float64 `json:"news_score"`
	RedditScore          float64 `json:"reddit_score"`
	SocialScore          float64 `json:"social_score"`
}

func NewSentimentSuggestion(
	sources *vo.SentimentSources,
	reasoning string,
	confidence float64,
) (*SentimentSuggestion, error) {
	if sources == nil {
		return nil, fmt.Errorf("sentiment sources cannot be nil")
	}
	
	if confidence < 0.0 || confidence > 1.0 {
		return nil, fmt.Errorf("confidence must be between 0.0 and 1.0, got: %.3f", confidence)
	}
	
	overallScore, err := sources.CalculateOverallScore()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate overall score: %w", err)
	}
	
	level := overallScore.GetLevel()
	
	// Generate suggestions based on sentiment level
	multiplier, threshold, interval := generateSuggestions(level)
	
	return &SentimentSuggestion{
		id:                  vo.NewEntityId(),
		overallScore:        overallScore,
		level:               level,
		sources:             sources,
		suggestedMultiplier: multiplier,
		suggestedThreshold:  threshold,
		suggestedInterval:   interval,
		reasoning:           reasoning,
		confidence:          confidence,
		status:              StatusPending,
		createdAt:           time.Now(),
	}, nil
}

func generateSuggestions(level vo.SentimentLevel) (multiplier, threshold float64, interval int) {
	switch level {
	case vo.VeryBullish:
		return 1.5, 0.8, 300  // +50% exposure, 0.8% profit, 5min
	case vo.Bullish:
		return 1.2, 1.0, 600  // +20% exposure, 1.0% profit, 10min
	case vo.Neutral:
		return 1.0, 1.5, 900  // Normal exposure, 1.5% profit, 15min
	case vo.Bearish:
		return 0.7, 2.0, 1800 // -30% exposure, 2.0% profit, 30min
	case vo.VeryBearish:
		return 0.4, 3.0, 3600 // -60% exposure, 3.0% profit, 1h
	default:
		return 1.0, 1.5, 900  // Default to neutral
	}
}

func (s *SentimentSuggestion) Approve(notes string) error {
	if s.status != StatusPending {
		return fmt.Errorf("can only approve pending suggestions, current status: %s", s.status)
	}
	
	s.status = StatusApproved
	s.userNotes = notes
	s.appliedMultiplier = &s.suggestedMultiplier
	s.appliedThreshold = &s.suggestedThreshold
	s.appliedInterval = &s.suggestedInterval
	now := time.Now()
	s.respondedAt = &now
	
	return nil
}

func (s *SentimentSuggestion) Ignore(notes string) error {
	if s.status != StatusPending {
		return fmt.Errorf("can only ignore pending suggestions, current status: %s", s.status)
	}
	
	s.status = StatusIgnored
	s.userNotes = notes
	now := time.Now()
	s.respondedAt = &now
	
	return nil
}

func (s *SentimentSuggestion) Customize(multiplier, threshold float64, interval int, notes string) error {
	if s.status != StatusPending {
		return fmt.Errorf("can only customize pending suggestions, current status: %s", s.status)
	}
	
	if multiplier <= 0 {
		return fmt.Errorf("multiplier must be positive, got: %.3f", multiplier)
	}
	
	if threshold <= 0 {
		return fmt.Errorf("threshold must be positive, got: %.3f", threshold)
	}
	
	if interval <= 0 {
		return fmt.Errorf("interval must be positive, got: %d", interval)
	}
	
	s.status = StatusCustomized
	s.userNotes = notes
	s.appliedMultiplier = &multiplier
	s.appliedThreshold = &threshold
	s.appliedInterval = &interval
	now := time.Now()
	s.respondedAt = &now
	
	return nil
}

func (s *SentimentSuggestion) ToDTO() SentimentSuggestionDTO {
	return SentimentSuggestionDTO{
		Id:                   s.id.GetValue(),
		OverallScore:         s.overallScore.GetValue(),
		Level:                s.level.GetValue(),
		LevelDisplay:         s.level.GetDisplayName(),
		Sources: SentimentSourcesDTO{
			FearGreedIndex:          s.sources.GetFearGreedIndex(),
			FearGreedClassification: s.sources.GetFearGreedClassification(),
			NewsScore:               s.sources.GetNewsScore(),
			RedditScore:             s.sources.GetRedditScore(),
			SocialScore:             s.sources.GetSocialScore(),
		},
		SuggestedMultiplier: s.suggestedMultiplier,
		SuggestedThreshold:  s.suggestedThreshold,
		SuggestedInterval:   s.suggestedInterval,
		Reasoning:           s.reasoning,
		Confidence:          s.confidence,
		Status:              string(s.status),
		UserNotes:           s.userNotes,
		AppliedMultiplier:   s.appliedMultiplier,
		AppliedThreshold:    s.appliedThreshold,
		AppliedInterval:     s.appliedInterval,
		CreatedAt:           s.createdAt,
		RespondedAt:         s.respondedAt,
		ApprovalRequired:    s.status == StatusPending,
	}
}

// Getters
func (s *SentimentSuggestion) GetId() *vo.EntityId {
	return s.id
}

func (s *SentimentSuggestion) GetOverallScore() *vo.SentimentScore {
	return s.overallScore
}

func (s *SentimentSuggestion) GetLevel() vo.SentimentLevel {
	return s.level
}

func (s *SentimentSuggestion) GetSources() *vo.SentimentSources {
	return s.sources
}

func (s *SentimentSuggestion) GetStatus() SuggestionStatus {
	return s.status
}

func (s *SentimentSuggestion) IsPending() bool {
	return s.status == StatusPending
}

func (s *SentimentSuggestion) IsApplied() bool {
	return s.status == StatusApproved || s.status == StatusCustomized
}

func (s *SentimentSuggestion) GetAppliedValues() (multiplier, threshold float64, interval int, ok bool) {
	if !s.IsApplied() {
		return 0, 0, 0, false
	}
	
	return *s.appliedMultiplier, *s.appliedThreshold, *s.appliedInterval, true
}

// Repository reconstruction helpers
// These methods are used exclusively by the repository layer to reconstruct entities from the database
// They should NOT be used in normal business logic

func (s *SentimentSuggestion) SetIdForReconstruction(id *vo.EntityId) {
	s.id = id
}

func (s *SentimentSuggestion) SetStatusForReconstruction(status SuggestionStatus) {
	s.status = status
}

func (s *SentimentSuggestion) SetUserNotesForReconstruction(notes string) {
	s.userNotes = notes
}

func (s *SentimentSuggestion) SetTimestampsForReconstruction(createdAt time.Time, respondedAt *time.Time) {
	s.createdAt = createdAt
	s.respondedAt = respondedAt
}

func (s *SentimentSuggestion) SetAppliedValuesForReconstruction(multiplier, threshold *float64, interval *int) {
	s.appliedMultiplier = multiplier
	s.appliedThreshold = threshold
	s.appliedInterval = interval
}
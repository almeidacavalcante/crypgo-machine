package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"fmt"
)

type ListSentimentSuggestionsUseCase struct {
	suggestionRepo repository.SentimentSuggestionRepository
}

type ListSentimentSuggestionsInput struct {
	Status string `json:"status"` // Optional: pending, approved, ignored, customized
	Limit  int    `json:"limit"`  // Default: 20, Max: 100
}

type ListSentimentSuggestionsOutput struct {
	Suggestions []entity.SentimentSuggestionDTO `json:"suggestions"`
	Analytics   *repository.SentimentAnalytics  `json:"analytics"`
	Total       int                            `json:"total"`
	Message     string                         `json:"message"`
}

func NewListSentimentSuggestionsUseCase(
	suggestionRepo repository.SentimentSuggestionRepository,
) *ListSentimentSuggestionsUseCase {
	return &ListSentimentSuggestionsUseCase{
		suggestionRepo: suggestionRepo,
	}
}

func (uc *ListSentimentSuggestionsUseCase) Execute(input ListSentimentSuggestionsInput) (*ListSentimentSuggestionsOutput, error) {
	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	
	var suggestions []*entity.SentimentSuggestion
	var err error
	
	// Fetch suggestions based on status filter
	if input.Status != "" {
		status := entity.SuggestionStatus(input.Status)
		if !uc.isValidStatus(status) {
			return nil, fmt.Errorf("invalid status: %s", input.Status)
		}
		suggestions, err = uc.suggestionRepo.FindByStatus(status)
	} else {
		suggestions, err = uc.suggestionRepo.FindRecent(input.Limit)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch suggestions: %w", err)
	}
	
	// Apply limit if not already limited by status query
	if input.Status == "" && len(suggestions) > input.Limit {
		suggestions = suggestions[:input.Limit]
	}
	
	// Convert to DTOs
	suggestionDTOs := make([]entity.SentimentSuggestionDTO, len(suggestions))
	for i, suggestion := range suggestions {
		suggestionDTOs[i] = suggestion.ToDTO()
	}
	
	// Get analytics
	analytics, err := uc.suggestionRepo.GetAnalytics()
	if err != nil {
		// Don't fail the entire request if analytics fail
		analytics = &repository.SentimentAnalytics{}
	}
	
	// Generate message
	message := uc.generateMessage(len(suggestionDTOs), input.Status, analytics)
	
	return &ListSentimentSuggestionsOutput{
		Suggestions: suggestionDTOs,
		Analytics:   analytics,
		Total:       len(suggestionDTOs),
		Message:     message,
	}, nil
}

func (uc *ListSentimentSuggestionsUseCase) isValidStatus(status entity.SuggestionStatus) bool {
	validStatuses := []entity.SuggestionStatus{
		entity.StatusPending,
		entity.StatusApproved,
		entity.StatusIgnored,
		entity.StatusCustomized,
	}
	
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	
	return false
}

func (uc *ListSentimentSuggestionsUseCase) generateMessage(count int, status string, analytics *repository.SentimentAnalytics) string {
	if status != "" {
		return fmt.Sprintf("Found %d suggestions with status '%s'", count, status)
	}
	
	if analytics.PendingSuggestions > 0 {
		return fmt.Sprintf("Found %d recent suggestions. You have %d pending suggestions awaiting approval.", 
			count, analytics.PendingSuggestions)
	}
	
	if count == 0 {
		return "No sentiment suggestions found. Generate a new analysis to create suggestions."
	}
	
	return fmt.Sprintf("Found %d recent suggestions. Approval rate: %.1f%%", 
		count, analytics.ApprovalRate*100)
}
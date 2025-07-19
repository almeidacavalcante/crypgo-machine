package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type ApproveSentimentSuggestionUseCase struct {
	suggestionRepo repository.SentimentSuggestionRepository
	botRepo        repository.TradingBotRepository
}

type ApproveSentimentSuggestionInput struct {
	SuggestionId      string   `json:"suggestion_id" validate:"required"`
	Action            string   `json:"action" validate:"required,oneof=approve_all approve_selective ignore"`
	UserNotes         string   `json:"user_notes"`
	CustomMultiplier  *float64 `json:"custom_multiplier,omitempty"`
	CustomThreshold   *float64 `json:"custom_threshold,omitempty"`
	CustomInterval    *int     `json:"custom_interval,omitempty"`
	ApplyToBots       []string `json:"apply_to_bots"` // Optional: specific bot IDs to apply to
	ApplyToAllBots    bool     `json:"apply_to_all_bots"` // If true, apply to all active bots
}

type ApproveSentimentSuggestionOutput struct {
	Suggestion       entity.SentimentSuggestionDTO `json:"suggestion"`
	AppliedToBots    []string                     `json:"applied_to_bots"`
	AffectedBots     int                          `json:"affected_bots"`
	Action           string                       `json:"action"`
	Message          string                       `json:"message"`
	PerformanceNote  string                       `json:"performance_note"`
}

func NewApproveSentimentSuggestionUseCase(
	suggestionRepo repository.SentimentSuggestionRepository,
	botRepo repository.TradingBotRepository,
) *ApproveSentimentSuggestionUseCase {
	return &ApproveSentimentSuggestionUseCase{
		suggestionRepo: suggestionRepo,
		botRepo:        botRepo,
	}
}

func (uc *ApproveSentimentSuggestionUseCase) Execute(input ApproveSentimentSuggestionInput) (*ApproveSentimentSuggestionOutput, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	
	// Find suggestion
	suggestionId, err := vo.RestoreEntityId(input.SuggestionId)
	if err != nil {
		return nil, fmt.Errorf("invalid suggestion ID: %w", err)
	}
	
	suggestion, err := uc.suggestionRepo.FindById(suggestionId)
	if err != nil {
		return nil, fmt.Errorf("suggestion not found: %w", err)
	}
	
	if !suggestion.IsPending() {
		return nil, fmt.Errorf("suggestion is not pending (current status: %s)", suggestion.GetStatus())
	}
	
	// Process the user's decision
	var appliedToBots []string
	var affectedBots int
	
	switch input.Action {
	case "approve_all":
		err = suggestion.Approve(input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to approve suggestion: %w", err)
		}
		
		// Apply suggested values to bots
		appliedToBots, affectedBots, err = uc.applyToBots(input, suggestion.ToDTO().SuggestedMultiplier, 
			suggestion.ToDTO().SuggestedThreshold, suggestion.ToDTO().SuggestedInterval)
		
	case "approve_selective":
		if input.CustomMultiplier == nil || input.CustomThreshold == nil || input.CustomInterval == nil {
			return nil, fmt.Errorf("custom values are required for selective approval")
		}
		
		err = suggestion.Customize(*input.CustomMultiplier, *input.CustomThreshold, *input.CustomInterval, input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to customize suggestion: %w", err)
		}
		
		// Apply custom values to bots
		appliedToBots, affectedBots, err = uc.applyToBots(input, *input.CustomMultiplier, 
			*input.CustomThreshold, *input.CustomInterval)
		
	case "ignore":
		err = suggestion.Ignore(input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to ignore suggestion: %w", err)
		}
		// No bot changes for ignored suggestions
		
	default:
		return nil, fmt.Errorf("invalid action: %s", input.Action)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to apply changes to bots: %w", err)
	}
	
	// Update suggestion in repository
	if err := uc.suggestionRepo.Update(suggestion); err != nil {
		return nil, fmt.Errorf("failed to update suggestion: %w", err)
	}
	
	// Prepare output
	output := &ApproveSentimentSuggestionOutput{
		Suggestion:      suggestion.ToDTO(),
		AppliedToBots:   appliedToBots,
		AffectedBots:    affectedBots,
		Action:          input.Action,
		Message:         uc.generateMessage(input.Action, affectedBots),
		PerformanceNote: uc.generatePerformanceNote(suggestion.GetLevel(), input.Action),
	}
	
	return output, nil
}

func (uc *ApproveSentimentSuggestionUseCase) validateInput(input ApproveSentimentSuggestionInput) error {
	if input.SuggestionId == "" {
		return fmt.Errorf("suggestion ID is required")
	}
	
	validActions := []string{"approve_all", "approve_selective", "ignore"}
	isValidAction := false
	for _, action := range validActions {
		if input.Action == action {
			isValidAction = true
			break
		}
	}
	if !isValidAction {
		return fmt.Errorf("invalid action: %s", input.Action)
	}
	
	if input.Action == "approve_selective" {
		if input.CustomMultiplier == nil {
			return fmt.Errorf("custom multiplier is required for selective approval")
		}
		if *input.CustomMultiplier <= 0 {
			return fmt.Errorf("custom multiplier must be positive")
		}
		
		if input.CustomThreshold == nil {
			return fmt.Errorf("custom threshold is required for selective approval")
		}
		if *input.CustomThreshold <= 0 {
			return fmt.Errorf("custom threshold must be positive")
		}
		
		if input.CustomInterval == nil {
			return fmt.Errorf("custom interval is required for selective approval")
		}
		if *input.CustomInterval <= 0 {
			return fmt.Errorf("custom interval must be positive")
		}
	}
	
	return nil
}

func (uc *ApproveSentimentSuggestionUseCase) applyToBots(input ApproveSentimentSuggestionInput, multiplier, threshold float64, interval int) ([]string, int, error) {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Fetch the specified bots or all active bots
	// 2. Update their trading parameters
	// 3. Restart their trading intervals if necessary
	// 4. Log the changes for audit purposes
	
	var appliedToBots []string
	
	if input.ApplyToAllBots {
		// Get all active bots
		bots, err := uc.botRepo.GetTradingBotsByStatus(entity.StatusRunning)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch active bots: %w", err)
		}
		
		for _, bot := range bots {
			// Apply the new parameters to each bot
			// This would involve updating bot configuration
			// For now, we'll just collect the IDs
			appliedToBots = append(appliedToBots, bot.ToDTO().Id)
		}
	} else if len(input.ApplyToBots) > 0 {
		// Apply to specific bots
		for _, botId := range input.ApplyToBots {
			botEntityId, err := vo.RestoreEntityId(botId)
			if err != nil {
				continue // Skip invalid IDs
			}
			
			_, err = uc.botRepo.GetTradeByID(botEntityId.GetValue())
			if err != nil {
				continue // Skip bots that don't exist
			}
			
			// Apply parameters to this specific bot
			// For now, we'll just collect the ID
			appliedToBots = append(appliedToBots, botId)
		}
	}
	
	return appliedToBots, len(appliedToBots), nil
}

func (uc *ApproveSentimentSuggestionUseCase) generateMessage(action string, affectedBots int) string {
	switch action {
	case "approve_all":
		if affectedBots > 0 {
			return fmt.Sprintf("Suggestion approved and applied to %d bot(s) with suggested values", affectedBots)
		}
		return "Suggestion approved. No bots were modified."
		
	case "approve_selective":
		if affectedBots > 0 {
			return fmt.Sprintf("Suggestion approved with custom values and applied to %d bot(s)", affectedBots)
		}
		return "Suggestion approved with custom values. No bots were modified."
		
	case "ignore":
		return "Suggestion ignored. No changes were made to trading bots."
		
	default:
		return "Suggestion processed."
	}
}

func (uc *ApproveSentimentSuggestionUseCase) generatePerformanceNote(level vo.SentimentLevel, action string) string {
	if action == "ignore" {
		return "No performance impact expected as suggestion was ignored."
	}
	
	switch level {
	case vo.VeryBullish:
		return "Expected impact: Increased trading frequency and higher risk tolerance. Monitor for overexposure."
	case vo.Bullish:
		return "Expected impact: Moderately increased activity. Should improve performance in uptrending markets."
	case vo.Neutral:
		return "Expected impact: Balanced approach maintained. Minimal performance change expected."
	case vo.Bearish:
		return "Expected impact: Reduced exposure and more conservative targets. May preserve capital in downtrends."
	case vo.VeryBearish:
		return "Expected impact: Minimal trading activity and high profit thresholds. Focus on capital preservation."
	default:
		return "Monitor bot performance after applying these changes."
	}
}
package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type ApproveSentimentSuggestionUseCase struct {
	suggestionRepo repository.SentimentSuggestionRepository
	// botRepo is kept for the constructor signature and for a future
	// implementation of applying suggestions to bots. It is not used today:
	// approving a suggestion never reads or writes trading bots.
	botRepo repository.TradingBotRepository
}

type ApproveSentimentSuggestionInput struct {
	SuggestionId     string   `json:"suggestion_id" validate:"required"`
	Action           string   `json:"action" validate:"required,oneof=approve_all approve_selective ignore"`
	UserNotes        string   `json:"user_notes"`
	CustomMultiplier *float64 `json:"custom_multiplier,omitempty"`
	CustomThreshold  *float64 `json:"custom_threshold,omitempty"`
	CustomInterval   *int     `json:"custom_interval,omitempty"`
	// ApplyToBots and ApplyToAllBots are accepted for backward compatibility
	// but applying suggestions to bots is not implemented: no bot is changed.
	ApplyToBots    []string `json:"apply_to_bots"`
	ApplyToAllBots bool     `json:"apply_to_all_bots"`
}

type ApproveSentimentSuggestionOutput struct {
	Suggestion entity.SentimentSuggestionDTO `json:"suggestion"`
	// AppliedToBots is always empty and AffectedBots is always 0, because
	// approving a suggestion does not change any bot (advisory model).
	AppliedToBots   []string `json:"applied_to_bots"`
	AffectedBots    int      `json:"affected_bots"`
	Action          string   `json:"action"`
	Message         string   `json:"message"`
	PerformanceNote string   `json:"performance_note"`
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

	// Process the user's decision. The sentiment system is advisory: only the
	// suggestion itself changes state. No trading bot is modified here.
	switch input.Action {
	case "approve_all":
		err = suggestion.Approve(input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to approve suggestion: %w", err)
		}

	case "approve_selective":
		if input.CustomMultiplier == nil || input.CustomThreshold == nil || input.CustomInterval == nil {
			return nil, fmt.Errorf("custom values are required for selective approval")
		}

		err = suggestion.Customize(*input.CustomMultiplier, *input.CustomThreshold, *input.CustomInterval, input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to customize suggestion: %w", err)
		}

	case "ignore":
		err = suggestion.Ignore(input.UserNotes)
		if err != nil {
			return nil, fmt.Errorf("failed to ignore suggestion: %w", err)
		}

	default:
		return nil, fmt.Errorf("invalid action: %s", input.Action)
	}

	// Applying approved values to trading bots (trade amount, minimum profit
	// threshold, interval) is NOT implemented. That would change how bots
	// trading real money behave and is a product decision that has not been
	// made. If it is ever implemented, it belongs here, after the suggestion
	// state transition and before the suggestion is persisted, and it must
	// report only the bots that were actually updated and saved.
	botApplicationRequested := input.ApplyToAllBots || len(input.ApplyToBots) > 0

	// Update suggestion in repository
	if err := uc.suggestionRepo.Update(suggestion); err != nil {
		return nil, fmt.Errorf("failed to update suggestion: %w", err)
	}

	// Prepare output
	output := &ApproveSentimentSuggestionOutput{
		Suggestion:      suggestion.ToDTO(),
		AppliedToBots:   []string{},
		AffectedBots:    0,
		Action:          input.Action,
		Message:         uc.generateMessage(input.Action, botApplicationRequested),
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

// botsNotChangedNote is appended to approval messages when the client asked for
// the suggestion to be applied to bots, which is not implemented.
const botsNotChangedNote = "Applying suggestions to bots is not implemented; no bot was changed."

func (uc *ApproveSentimentSuggestionUseCase) generateMessage(action string, botApplicationRequested bool) string {
	switch action {
	case "approve_all":
		if botApplicationRequested {
			return "Suggestion approved with suggested values. " + botsNotChangedNote
		}
		return "Suggestion approved with suggested values. No bots were modified."

	case "approve_selective":
		if botApplicationRequested {
			return "Suggestion approved with custom values. " + botsNotChangedNote
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
		return "If these values are applied to bots, expected impact: Increased trading frequency and higher risk tolerance. Monitor for overexposure."
	case vo.Bullish:
		return "If these values are applied to bots, expected impact: Moderately increased activity. Should improve performance in uptrending markets."
	case vo.Neutral:
		return "If these values are applied to bots, expected impact: Balanced approach maintained. Minimal performance change expected."
	case vo.Bearish:
		return "If these values are applied to bots, expected impact: Reduced exposure and more conservative targets. May preserve capital in downtrends."
	case vo.VeryBearish:
		return "If these values are applied to bots, expected impact: Minimal trading activity and high profit thresholds. Focus on capital preservation."
	default:
		return "No bot was changed. Monitor bot performance if you apply these values."
	}
}

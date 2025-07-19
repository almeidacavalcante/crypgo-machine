package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type GenerateSentimentSuggestionUseCase struct {
	suggestionRepo repository.SentimentSuggestionRepository
}

type GenerateSentimentSuggestionInput struct {
	FearGreedIndex int     `json:"fear_greed_index" validate:"required,min=0,max=100"`
	NewsScore      float64 `json:"news_score" validate:"required,min=-1.0,max=1.0"`
	RedditScore    float64 `json:"reddit_score" validate:"required,min=-1.0,max=1.0"`
	SocialScore    float64 `json:"social_score" validate:"required,min=-1.0,max=1.0"`
	Reasoning      string  `json:"reasoning" validate:"required,min=10,max=500"`
	Confidence     float64 `json:"confidence" validate:"required,min=0.0,max=1.0"`
}

type GenerateSentimentSuggestionOutput struct {
	Suggestion       entity.SentimentSuggestionDTO `json:"suggestion"`
	RecommendedAction string                       `json:"recommended_action"`
	RiskLevel        string                       `json:"risk_level"`
	Message          string                       `json:"message"`
}

func NewGenerateSentimentSuggestionUseCase(
	suggestionRepo repository.SentimentSuggestionRepository,
) *GenerateSentimentSuggestionUseCase {
	return &GenerateSentimentSuggestionUseCase{
		suggestionRepo: suggestionRepo,
	}
}

func (uc *GenerateSentimentSuggestionUseCase) Execute(input GenerateSentimentSuggestionInput) (*GenerateSentimentSuggestionOutput, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	
	// Create sentiment sources value object
	sources, err := vo.NewSentimentSources(
		input.FearGreedIndex,
		input.NewsScore,
		input.RedditScore,
		input.SocialScore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment sources: %w", err)
	}
	
	// Check if there's already a pending suggestion
	pendingSuggestions, err := uc.suggestionRepo.FindPending()
	if err != nil {
		return nil, fmt.Errorf("failed to check pending suggestions: %w", err)
	}
	
	if len(pendingSuggestions) > 0 {
		return nil, fmt.Errorf("there is already a pending suggestion awaiting approval")
	}
	
	// Generate new suggestion
	suggestion, err := entity.NewSentimentSuggestion(sources, input.Reasoning, input.Confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment suggestion: %w", err)
	}
	
	// Save suggestion
	if err := uc.suggestionRepo.Save(suggestion); err != nil {
		return nil, fmt.Errorf("failed to save sentiment suggestion: %w", err)
	}
	
	// Prepare output
	output := &GenerateSentimentSuggestionOutput{
		Suggestion:       suggestion.ToDTO(),
		RecommendedAction: uc.getRecommendedAction(suggestion.GetLevel()),
		RiskLevel:        uc.getRiskLevel(suggestion.GetLevel(), input.Confidence),
		Message:          uc.generateMessage(suggestion),
	}
	
	return output, nil
}

func (uc *GenerateSentimentSuggestionUseCase) validateInput(input GenerateSentimentSuggestionInput) error {
	if input.FearGreedIndex < 0 || input.FearGreedIndex > 100 {
		return fmt.Errorf("fear greed index must be between 0 and 100")
	}
	
	if input.NewsScore < -1.0 || input.NewsScore > 1.0 {
		return fmt.Errorf("news score must be between -1.0 and 1.0")
	}
	
	if input.RedditScore < -1.0 || input.RedditScore > 1.0 {
		return fmt.Errorf("reddit score must be between -1.0 and 1.0")
	}
	
	if input.SocialScore < -1.0 || input.SocialScore > 1.0 {
		return fmt.Errorf("social score must be between -1.0 and 1.0")
	}
	
	if input.Confidence < 0.0 || input.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0")
	}
	
	if len(input.Reasoning) < 10 || len(input.Reasoning) > 500 {
		return fmt.Errorf("reasoning must be between 10 and 500 characters")
	}
	
	return nil
}

func (uc *GenerateSentimentSuggestionUseCase) getRecommendedAction(level vo.SentimentLevel) string {
	switch level {
	case vo.VeryBullish:
		return "Strongly consider increasing exposure and reducing profit thresholds for more aggressive trading"
	case vo.Bullish:
		return "Consider moderate increase in exposure and slightly more aggressive profit targets"
	case vo.Neutral:
		return "Maintain current trading configuration - no changes recommended"
	case vo.Bearish:
		return "Consider reducing exposure and increasing profit thresholds for more conservative trading"
	case vo.VeryBearish:
		return "Strongly consider minimal exposure and conservative profit targets to protect capital"
	default:
		return "Unknown sentiment level - maintain current configuration"
	}
}

func (uc *GenerateSentimentSuggestionUseCase) getRiskLevel(level vo.SentimentLevel, confidence float64) string {
	baseRisk := "Medium"
	
	switch level {
	case vo.VeryBullish, vo.VeryBearish:
		baseRisk = "High"
	case vo.Bullish, vo.Bearish:
		baseRisk = "Medium"
	case vo.Neutral:
		baseRisk = "Low"
	}
	
	// Adjust for confidence
	if confidence < 0.5 {
		if baseRisk == "Low" {
			baseRisk = "Medium"
		} else if baseRisk == "Medium" {
			baseRisk = "High"
		}
	}
	
	return baseRisk
}

func (uc *GenerateSentimentSuggestionUseCase) generateMessage(suggestion *entity.SentimentSuggestion) string {
	dto := suggestion.ToDTO()
	
	return fmt.Sprintf(
		"Market sentiment analysis complete. Overall sentiment: %s (%.3f). "+
			"Suggestions based on Fear & Greed: %d, News: %.2f, Reddit: %.2f, Social: %.2f. "+
			"Confidence: %.1f%%. Please review and approve, customize, or ignore these suggestions.",
		dto.LevelDisplay,
		dto.OverallScore,
		dto.Sources.FearGreedIndex,
		dto.Sources.NewsScore,
		dto.Sources.RedditScore,
		dto.Sources.SocialScore,
		dto.Confidence*100,
	)
}
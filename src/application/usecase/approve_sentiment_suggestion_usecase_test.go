package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
	"strings"
	"testing"
)

// sentimentApproveFakeSuggestionRepo is an in-memory SentimentSuggestionRepository
// used only by the approve sentiment suggestion tests.
type sentimentApproveFakeSuggestionRepo struct {
	byId        map[string]*entity.SentimentSuggestion
	updateCalls int
	lastUpdated *entity.SentimentSuggestion
}

func newSentimentApproveFakeSuggestionRepo(suggestions ...*entity.SentimentSuggestion) *sentimentApproveFakeSuggestionRepo {
	repo := &sentimentApproveFakeSuggestionRepo{byId: map[string]*entity.SentimentSuggestion{}}
	for _, s := range suggestions {
		repo.byId[s.GetId().GetValue()] = s
	}
	return repo
}

func (r *sentimentApproveFakeSuggestionRepo) Save(s *entity.SentimentSuggestion) error {
	r.byId[s.GetId().GetValue()] = s
	return nil
}

func (r *sentimentApproveFakeSuggestionRepo) FindById(id *vo.EntityId) (*entity.SentimentSuggestion, error) {
	s, ok := r.byId[id.GetValue()]
	if !ok {
		return nil, fmt.Errorf("suggestion %s not found", id.GetValue())
	}
	return s, nil
}

func (r *sentimentApproveFakeSuggestionRepo) FindPending() ([]*entity.SentimentSuggestion, error) {
	return nil, nil
}

func (r *sentimentApproveFakeSuggestionRepo) FindByStatus(entity.SuggestionStatus) ([]*entity.SentimentSuggestion, error) {
	return nil, nil
}

func (r *sentimentApproveFakeSuggestionRepo) FindRecent(int) ([]*entity.SentimentSuggestion, error) {
	return nil, nil
}

func (r *sentimentApproveFakeSuggestionRepo) FindByDateRange(string, string) ([]*entity.SentimentSuggestion, error) {
	return nil, nil
}

func (r *sentimentApproveFakeSuggestionRepo) Update(s *entity.SentimentSuggestion) error {
	r.updateCalls++
	r.lastUpdated = s
	r.byId[s.GetId().GetValue()] = s
	return nil
}

func (r *sentimentApproveFakeSuggestionRepo) GetAnalytics() (*repository.SentimentAnalytics, error) {
	return &repository.SentimentAnalytics{}, nil
}

func (r *sentimentApproveFakeSuggestionRepo) DeleteOlderThan(int) error {
	return nil
}

// sentimentApproveSpyBotRepo is a TradingBotRepository that records every call.
// Approving a sentiment suggestion must not touch any bot.
type sentimentApproveSpyBotRepo struct {
	runningBots []*entity.TradingBot
	writes      int
	reads       int
}

func (r *sentimentApproveSpyBotRepo) Save(*entity.TradingBot) error {
	r.writes++
	return nil
}

func (r *sentimentApproveSpyBotRepo) Update(*entity.TradingBot) error {
	r.writes++
	return nil
}

func (r *sentimentApproveSpyBotRepo) GetTradeByID(string) (*entity.TradingBot, error) {
	r.reads++
	return nil, fmt.Errorf("not found")
}

func (r *sentimentApproveSpyBotRepo) Exists(string) (bool, error) {
	r.reads++
	return false, nil
}

func (r *sentimentApproveSpyBotRepo) GetAllTradingBots() ([]*entity.TradingBot, error) {
	r.reads++
	return r.runningBots, nil
}

func (r *sentimentApproveSpyBotRepo) GetTradingBotsByStatus(entity.Status) ([]*entity.TradingBot, error) {
	r.reads++
	return r.runningBots, nil
}

func newPendingSentimentSuggestion(t *testing.T) *entity.SentimentSuggestion {
	t.Helper()
	sources, err := vo.NewSentimentSources(80, 0.6, 0.5, 0.4)
	if err != nil {
		t.Fatalf("failed to create sentiment sources: %v", err)
	}
	suggestion, err := entity.NewSentimentSuggestion(sources, "test reasoning", 0.8)
	if err != nil {
		t.Fatalf("failed to create sentiment suggestion: %v", err)
	}
	return suggestion
}

func assertNoBotReportedAsChanged(t *testing.T, output *ApproveSentimentSuggestionOutput, botRepo *sentimentApproveSpyBotRepo) {
	t.Helper()
	if botRepo.writes != 0 {
		t.Errorf("expected no bot repository writes, got %d", botRepo.writes)
	}
	if botRepo.reads != 0 {
		t.Errorf("expected no bot repository reads, got %d", botRepo.reads)
	}
	if output.AffectedBots != 0 {
		t.Errorf("expected affected_bots = 0, got %d", output.AffectedBots)
	}
	if output.AppliedToBots == nil || len(output.AppliedToBots) != 0 {
		t.Errorf("expected applied_to_bots to be an empty list, got %v", output.AppliedToBots)
	}
	if strings.Contains(output.Message, "applied to") {
		t.Errorf("message must not claim the suggestion was applied to bots: %q", output.Message)
	}
}

func TestApproveSentimentSuggestion_ApproveAllWithApplyToAllBotsDoesNotTouchBots(t *testing.T) {
	suggestion := newPendingSentimentSuggestion(t)
	suggestionRepo := newSentimentApproveFakeSuggestionRepo(suggestion)
	// Pretend there are running bots so the old placeholder would have counted them.
	botRepo := &sentimentApproveSpyBotRepo{runningBots: []*entity.TradingBot{{}, {}}}
	uc := NewApproveSentimentSuggestionUseCase(suggestionRepo, botRepo)

	output, err := uc.Execute(ApproveSentimentSuggestionInput{
		SuggestionId:   suggestion.GetId().GetValue(),
		Action:         "approve_all",
		UserNotes:      "looks good",
		ApplyToAllBots: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suggestionRepo.updateCalls != 1 {
		t.Fatalf("expected suggestion to be persisted once, got %d", suggestionRepo.updateCalls)
	}
	if suggestionRepo.lastUpdated.GetStatus() != entity.StatusApproved {
		t.Errorf("expected status %s, got %s", entity.StatusApproved, suggestionRepo.lastUpdated.GetStatus())
	}
	dto := output.Suggestion
	if dto.AppliedMultiplier == nil || *dto.AppliedMultiplier != dto.SuggestedMultiplier {
		t.Errorf("expected applied multiplier to equal suggested multiplier %v, got %v", dto.SuggestedMultiplier, dto.AppliedMultiplier)
	}
	if dto.UserNotes != "looks good" {
		t.Errorf("expected user notes to be recorded, got %q", dto.UserNotes)
	}

	assertNoBotReportedAsChanged(t, output, botRepo)
	if !strings.Contains(output.Message, "not implemented") || !strings.Contains(output.Message, "no bot was changed") {
		t.Errorf("expected message to say applying to bots is not implemented, got %q", output.Message)
	}
	if output.Action != "approve_all" {
		t.Errorf("expected action approve_all, got %q", output.Action)
	}
}

func TestApproveSentimentSuggestion_ApproveSelectiveWithSpecificBotsDoesNotTouchBots(t *testing.T) {
	suggestion := newPendingSentimentSuggestion(t)
	suggestionRepo := newSentimentApproveFakeSuggestionRepo(suggestion)
	botRepo := &sentimentApproveSpyBotRepo{}
	uc := NewApproveSentimentSuggestionUseCase(suggestionRepo, botRepo)

	multiplier, threshold, interval := 0.9, 2.5, 1200
	output, err := uc.Execute(ApproveSentimentSuggestionInput{
		SuggestionId:     suggestion.GetId().GetValue(),
		Action:           "approve_selective",
		CustomMultiplier: &multiplier,
		CustomThreshold:  &threshold,
		CustomInterval:   &interval,
		ApplyToBots:      []string{vo.NewEntityId().GetValue()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suggestionRepo.updateCalls != 1 {
		t.Fatalf("expected suggestion to be persisted once, got %d", suggestionRepo.updateCalls)
	}
	if suggestionRepo.lastUpdated.GetStatus() != entity.StatusCustomized {
		t.Errorf("expected status %s, got %s", entity.StatusCustomized, suggestionRepo.lastUpdated.GetStatus())
	}
	m, th, i, ok := suggestionRepo.lastUpdated.GetAppliedValues()
	if !ok || m != multiplier || th != threshold || i != interval {
		t.Errorf("expected applied values (%v, %v, %v), got (%v, %v, %v, ok=%v)", multiplier, threshold, interval, m, th, i, ok)
	}

	assertNoBotReportedAsChanged(t, output, botRepo)
	if !strings.Contains(output.Message, "not implemented") {
		t.Errorf("expected message to say applying to bots is not implemented, got %q", output.Message)
	}
}

func TestApproveSentimentSuggestion_ApproveWithoutBotTargetsSaysNoBotsModified(t *testing.T) {
	suggestion := newPendingSentimentSuggestion(t)
	suggestionRepo := newSentimentApproveFakeSuggestionRepo(suggestion)
	botRepo := &sentimentApproveSpyBotRepo{}
	uc := NewApproveSentimentSuggestionUseCase(suggestionRepo, botRepo)

	output, err := uc.Execute(ApproveSentimentSuggestionInput{
		SuggestionId: suggestion.GetId().GetValue(),
		Action:       "approve_all",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertNoBotReportedAsChanged(t, output, botRepo)
	if !strings.Contains(output.Message, "No bots were modified") {
		t.Errorf("expected message to say no bots were modified, got %q", output.Message)
	}
}

func TestApproveSentimentSuggestion_IgnoreDoesNotTouchBots(t *testing.T) {
	suggestion := newPendingSentimentSuggestion(t)
	suggestionRepo := newSentimentApproveFakeSuggestionRepo(suggestion)
	botRepo := &sentimentApproveSpyBotRepo{runningBots: []*entity.TradingBot{{}}}
	uc := NewApproveSentimentSuggestionUseCase(suggestionRepo, botRepo)

	output, err := uc.Execute(ApproveSentimentSuggestionInput{
		SuggestionId:   suggestion.GetId().GetValue(),
		Action:         "ignore",
		UserNotes:      "not now",
		ApplyToAllBots: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suggestionRepo.updateCalls != 1 {
		t.Fatalf("expected suggestion to be persisted once, got %d", suggestionRepo.updateCalls)
	}
	if suggestionRepo.lastUpdated.GetStatus() != entity.StatusIgnored {
		t.Errorf("expected status %s, got %s", entity.StatusIgnored, suggestionRepo.lastUpdated.GetStatus())
	}
	if _, _, _, ok := suggestionRepo.lastUpdated.GetAppliedValues(); ok {
		t.Errorf("expected no applied values on an ignored suggestion")
	}

	assertNoBotReportedAsChanged(t, output, botRepo)
	if output.Message != "Suggestion ignored. No changes were made to trading bots." {
		t.Errorf("unexpected message for ignore: %q", output.Message)
	}
}

package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"database/sql"
	"fmt"
	"time"
)

type SentimentSuggestionRepositoryDatabase struct {
	db *sql.DB
}

func NewSentimentSuggestionRepositoryDatabase(db *sql.DB) repository.SentimentSuggestionRepository {
	return &SentimentSuggestionRepositoryDatabase{
		db: db,
	}
}

func (r *SentimentSuggestionRepositoryDatabase) Save(suggestion *entity.SentimentSuggestion) error {
	query := `
		INSERT INTO sentiment_suggestions (
			id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
			suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	
	dto := suggestion.ToDTO()
	
	_, err := r.db.Exec(query,
		dto.Id,
		dto.OverallScore,
		dto.Level,
		dto.Sources.FearGreedIndex,
		dto.Sources.NewsScore,
		dto.Sources.RedditScore,
		dto.Sources.SocialScore,
		dto.SuggestedMultiplier,
		dto.SuggestedThreshold,
		dto.SuggestedInterval,
		dto.Reasoning,
		dto.Confidence,
		dto.Status,
		dto.CreatedAt,
	)
	
	return err
}

func (r *SentimentSuggestionRepositoryDatabase) FindById(id *vo.EntityId) (*entity.SentimentSuggestion, error) {
	query := `
		SELECT id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
			   suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence,
			   status, user_notes, applied_multiplier, applied_threshold, applied_interval,
			   created_at, responded_at
		FROM sentiment_suggestions 
		WHERE id = $1
	`
	
	row := r.db.QueryRow(query, id.GetValue())
	return r.scanSuggestion(row)
}

func (r *SentimentSuggestionRepositoryDatabase) FindPending() ([]*entity.SentimentSuggestion, error) {
	return r.FindByStatus(entity.StatusPending)
}

func (r *SentimentSuggestionRepositoryDatabase) FindByStatus(status entity.SuggestionStatus) ([]*entity.SentimentSuggestion, error) {
	query := `
		SELECT id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
			   suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence,
			   status, user_notes, applied_multiplier, applied_threshold, applied_interval,
			   created_at, responded_at
		FROM sentiment_suggestions 
		WHERE status = $1 
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var suggestions []*entity.SentimentSuggestion
	for rows.Next() {
		suggestion, err := r.scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

func (r *SentimentSuggestionRepositoryDatabase) FindRecent(limit int) ([]*entity.SentimentSuggestion, error) {
	query := `
		SELECT id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
			   suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence,
			   status, user_notes, applied_multiplier, applied_threshold, applied_interval,
			   created_at, responded_at
		FROM sentiment_suggestions 
		ORDER BY created_at DESC 
		LIMIT $1
	`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var suggestions []*entity.SentimentSuggestion
	for rows.Next() {
		suggestion, err := r.scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

func (r *SentimentSuggestionRepositoryDatabase) FindByDateRange(from, to string) ([]*entity.SentimentSuggestion, error) {
	query := `
		SELECT id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
			   suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence,
			   status, user_notes, applied_multiplier, applied_threshold, applied_interval,
			   created_at, responded_at
		FROM sentiment_suggestions 
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var suggestions []*entity.SentimentSuggestion
	for rows.Next() {
		suggestion, err := r.scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions, nil
}

func (r *SentimentSuggestionRepositoryDatabase) Update(suggestion *entity.SentimentSuggestion) error {
	query := `
		UPDATE sentiment_suggestions 
		SET status = $2, user_notes = $3, applied_multiplier = $4, 
		    applied_threshold = $5, applied_interval = $6, responded_at = $7
		WHERE id = $1
	`
	
	dto := suggestion.ToDTO()
	
	_, err := r.db.Exec(query,
		dto.Id,
		dto.Status,
		dto.UserNotes,
		dto.AppliedMultiplier,
		dto.AppliedThreshold,
		dto.AppliedInterval,
		dto.RespondedAt,
	)
	
	return err
}

func (r *SentimentSuggestionRepositoryDatabase) GetAnalytics() (*repository.SentimentAnalytics, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved,
			COUNT(CASE WHEN status = 'ignored' THEN 1 END) as ignored,
			COUNT(CASE WHEN status = 'customized' THEN 1 END) as customized,
			AVG(CASE 
				WHEN responded_at IS NOT NULL AND status != 'pending' 
				THEN EXTRACT(EPOCH FROM (responded_at - created_at)) / 3600 
			END) as avg_response_hours,
			MAX(created_at) as last_suggestion
		FROM sentiment_suggestions
	`
	
	row := r.db.QueryRow(query)
	
	var analytics repository.SentimentAnalytics
	var lastSuggestion sql.NullTime
	var avgResponseHours sql.NullFloat64
	
	err := row.Scan(
		&analytics.TotalSuggestions,
		&analytics.PendingSuggestions,
		&analytics.ApprovedSuggestions,
		&analytics.IgnoredSuggestions,
		&analytics.CustomizedSuggestions,
		&avgResponseHours,
		&lastSuggestion,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Calculate rates
	if analytics.TotalSuggestions > 0 {
		analytics.ApprovalRate = float64(analytics.ApprovedSuggestions+analytics.CustomizedSuggestions) / float64(analytics.TotalSuggestions)
		analytics.CustomizationRate = float64(analytics.CustomizedSuggestions) / float64(analytics.TotalSuggestions)
	}
	
	if avgResponseHours.Valid {
		analytics.AvgResponseTimeHours = avgResponseHours.Float64
	}
	
	if lastSuggestion.Valid {
		analytics.LastSuggestionDate = lastSuggestion.Time.Format("2006-01-02 15:04:05")
	}
	
	return &analytics, nil
}

func (r *SentimentSuggestionRepositoryDatabase) DeleteOlderThan(days int) error {
	query := `
		DELETE FROM sentiment_suggestions 
		WHERE created_at < NOW() - INTERVAL '%d days'
	`
	
	_, err := r.db.Exec(fmt.Sprintf(query, days))
	return err
}

func (r *SentimentSuggestionRepositoryDatabase) scanSuggestion(scanner interface{}) (*entity.SentimentSuggestion, error) {
	var (
		id, level, status, reasoning                             string
		overallScore, newsScore, redditScore, socialScore       float64
		suggestedMultiplier, suggestedThreshold, confidence      float64
		fearGreedIndex, suggestedInterval                        int
		userNotes                                                sql.NullString
		appliedMultiplier, appliedThreshold                      sql.NullFloat64
		appliedInterval                                          sql.NullInt64
		createdAt                                                time.Time
		respondedAt                                              sql.NullTime
	)
	
	var err error
	
	switch s := scanner.(type) {
	case *sql.Row:
		err = s.Scan(&id, &overallScore, &level, &fearGreedIndex, &newsScore, &redditScore, &socialScore,
			&suggestedMultiplier, &suggestedThreshold, &suggestedInterval, &reasoning, &confidence,
			&status, &userNotes, &appliedMultiplier, &appliedThreshold, &appliedInterval,
			&createdAt, &respondedAt)
	case *sql.Rows:
		err = s.Scan(&id, &overallScore, &level, &fearGreedIndex, &newsScore, &redditScore, &socialScore,
			&suggestedMultiplier, &suggestedThreshold, &suggestedInterval, &reasoning, &confidence,
			&status, &userNotes, &appliedMultiplier, &appliedThreshold, &appliedInterval,
			&createdAt, &respondedAt)
	default:
		return nil, fmt.Errorf("unsupported scanner type")
	}
	
	if err != nil {
		return nil, err
	}
	
	// Reconstruct value objects and entities
	entityId, err := vo.RestoreEntityId(id)
	if err != nil {
		return nil, err
	}
	
	// Create sentiment sources for entity reconstruction
	sources, err := vo.NewSentimentSources(fearGreedIndex, newsScore, redditScore, socialScore)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct sentiment sources: %w", err)
	}
	
	suggestion, err := entity.NewSentimentSuggestion(sources, reasoning, confidence)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct sentiment suggestion: %w", err)
	}
	
	// Manually set the ID and status for reconstructed entity
	suggestion.SetIdForReconstruction(entityId)
	suggestion.SetStatusForReconstruction(entity.SuggestionStatus(status))
	var respondedAtPtr *time.Time
	if respondedAt.Valid {
		respondedAtPtr = &respondedAt.Time
	}
	suggestion.SetTimestampsForReconstruction(createdAt, respondedAtPtr)
	
	if userNotes.Valid {
		suggestion.SetUserNotesForReconstruction(userNotes.String)
	}
	
	// Set applied values if they exist
	var multiplier, threshold *float64
	var interval *int
	if appliedMultiplier.Valid {
		multiplier = &appliedMultiplier.Float64
	}
	if appliedThreshold.Valid {
		threshold = &appliedThreshold.Float64
	}
	if appliedInterval.Valid {
		val := int(appliedInterval.Int64)
		interval = &val
	}
	suggestion.SetAppliedValuesForReconstruction(multiplier, threshold, interval)
	
	return suggestion, nil
}
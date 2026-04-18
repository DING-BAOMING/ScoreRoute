package repository

import (
	"ai-gateway/internal/model"
	"time"
)

type SampleRatingRepo struct{}

func NewSampleRatingRepo() *SampleRatingRepo {
	return &SampleRatingRepo{}
}

func (r *SampleRatingRepo) Upsert(rating *model.SampleRating) error {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	rating.ExpiresAt = expiresAt
	rating.AnalyzedAt = time.Now()

	_, err := DB.Exec(`
		INSERT OR REPLACE INTO sample_ratings 
		(model_key, score, tool_calling_score, completeness_score, context_understanding_score, error_handling_score, response_quality_score, analyzed_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rating.ModelKey, rating.Score, rating.ToolCallingScore, rating.CompletenessScore,
		rating.ContextUnderstandingScore, rating.ErrorHandlingScore, rating.ResponseQualityScore,
		rating.AnalyzedAt, rating.ExpiresAt)
	return err
}

func (r *SampleRatingRepo) GetByModelKey(modelKey string) (*model.SampleRating, error) {
	rating := &model.SampleRating{}
	err := DB.QueryRow(`
		SELECT id, model_key, score, tool_calling_score, completeness_score, context_understanding_score, 
		       error_handling_score, response_quality_score, analyzed_at, expires_at
		FROM sample_ratings WHERE model_key = ?
	`, modelKey).Scan(&rating.ID, &rating.ModelKey, &rating.Score, &rating.ToolCallingScore,
		&rating.CompletenessScore, &rating.ContextUnderstandingScore, &rating.ErrorHandlingScore,
		&rating.ResponseQualityScore, &rating.AnalyzedAt, &rating.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return rating, nil
}

func (r *SampleRatingRepo) List() ([]*model.SampleRating, error) {
	rows, err := DB.Query(`
		SELECT id, model_key, score, tool_calling_score, completeness_score, context_understanding_score, 
		       error_handling_score, response_quality_score, analyzed_at, expires_at
		FROM sample_ratings ORDER BY score DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*model.SampleRating
	for rows.Next() {
		r := &model.SampleRating{}
		if err := rows.Scan(&r.ID, &r.ModelKey, &r.Score, &r.ToolCallingScore, &r.CompletenessScore,
			&r.ContextUnderstandingScore, &r.ErrorHandlingScore, &r.ResponseQualityScore,
			&r.AnalyzedAt, &r.ExpiresAt); err != nil {
			continue
		}
		ratings = append(ratings, r)
	}
	return ratings, rows.Err()
}

func (r *SampleRatingRepo) GetAllAsMap() (map[string]*model.SampleRating, error) {
	ratings, err := r.List()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.SampleRating)
	for _, r := range ratings {
		result[r.ModelKey] = r
	}
	return result, nil
}

func (r *SampleRatingRepo) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM sample_ratings WHERE id = ?`, id)
	return err
}

func (r *SampleRatingRepo) DeleteExpired() (int64, error) {
	now := time.Now()
	result, err := DB.Exec(`DELETE FROM sample_ratings WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

package repository

import (
	"database/sql"
	"fmt"
	"time"

	"ai-gateway/internal/model"
)

type SampleRepo struct{}

func NewSampleRepo() *SampleRepo {
	return &SampleRepo{}
}

func (r *SampleRepo) Create(sample *model.Sample) error {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	sample.ExpiresAt = expiresAt
	sample.CreatedAt = time.Now()

	_, err := DB.Exec(
		`INSERT OR REPLACE INTO samples (model_key, request_content, response_content, token_count, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		sample.ModelKey, sample.RequestContent, sample.ResponseContent, sample.TokenCount, sample.CreatedAt, sample.ExpiresAt,
	)
	return err
}

func (r *SampleRepo) GetByModelKey(modelKey string) (*model.Sample, error) {
	sample := &model.Sample{}
	err := DB.QueryRow(
		`SELECT id, model_key, request_content, response_content, token_count, created_at, expires_at FROM samples WHERE model_key = ?`,
		modelKey,
	).Scan(&sample.ID, &sample.ModelKey, &sample.RequestContent, &sample.ResponseContent, &sample.TokenCount, &sample.CreatedAt, &sample.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sample, nil
}

func (r *SampleRepo) List() ([]*model.Sample, error) {
	rows, err := DB.Query(`SELECT id, model_key, request_content, response_content, token_count, created_at, expires_at FROM samples ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []*model.Sample
	for rows.Next() {
		s := &model.Sample{}
		if err := rows.Scan(&s.ID, &s.ModelKey, &s.RequestContent, &s.ResponseContent, &s.TokenCount, &s.CreatedAt, &s.ExpiresAt); err != nil {
			continue
		}
		samples = append(samples, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return samples, nil
}

func (r *SampleRepo) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM samples WHERE id = ?`, id)
	return err
}

func (r *SampleRepo) DeleteExpired() (int64, error) {
	now := time.Now()
	result, err := DB.Exec(`DELETE FROM samples WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

func (r *SampleRepo) Exists(modelKey string) (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM samples WHERE model_key = ?`, modelKey).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SampleRepo) SaveSample(modelKey, requestContent, responseContent string, tokenCount int) error {
	if tokenCount < 1000 {
		return nil
	}

	exists, err := r.Exists(modelKey)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	sample := &model.Sample{
		ModelKey:       modelKey,
		RequestContent: requestContent,
		ResponseContent: responseContent,
		TokenCount:     tokenCount,
	}
	return r.Create(sample)
}

func (r *SampleRepo) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalSamples int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM samples`).Scan(&totalSamples); err != nil {
		totalSamples = 0
	}
	stats["total_samples"] = totalSamples

	var avgTokens int64
	if err := DB.QueryRow(`SELECT COALESCE(AVG(token_count), 0) FROM samples`).Scan(&avgTokens); err != nil {
		avgTokens = 0
	}
	stats["avg_tokens"] = avgTokens

	var modelsCount int64
	if err := DB.QueryRow(`SELECT COUNT(DISTINCT model_key) FROM samples`).Scan(&modelsCount); err != nil {
		modelsCount = 0
	}
	stats["models"] = modelsCount

	var expiredCount int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM samples WHERE expires_at < datetime('now')`).Scan(&expiredCount); err != nil {
		expiredCount = 0
	}
	stats["expired"] = expiredCount

	return stats, nil
}

func fmtTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func (r *SampleRepo) CleanupOlderThan(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := DB.Exec(`DELETE FROM samples WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup samples: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}
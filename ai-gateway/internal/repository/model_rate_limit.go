package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type ModelRateLimitUsage struct {
	ID           int64
	ModelID      int64
	RuleIndex    int
	CurrentCount int64
	WindowStart  time.Time
}

type ModelRateLimitRepo struct{}

func NewModelRateLimitRepo() *ModelRateLimitRepo {
	return &ModelRateLimitRepo{}
}

func (r *ModelRateLimitRepo) GetUsage(modelID int64, ruleIndex int) (*ModelRateLimitUsage, error) {
	usage := &ModelRateLimitUsage{}
	err := DB.QueryRow(
		`SELECT id, model_id, rule_index, current_count, window_start FROM model_rate_limit_usage WHERE model_id = ? AND rule_index = ?`,
		modelID, ruleIndex,
	).Scan(&usage.ID, &usage.ModelID, &usage.RuleIndex, &usage.CurrentCount, &usage.WindowStart)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model rate limit usage: %w", err)
	}
	return usage, nil
}

func (r *ModelRateLimitRepo) UpsertUsage(modelID int64, ruleIndex int, currentCount int64, windowStart time.Time, resetWindow bool) error {
	if resetWindow {
		_, err := DB.Exec(
			`INSERT INTO model_rate_limit_usage (model_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(model_id, rule_index) DO UPDATE SET 
			 current_count = ?, window_start = ?, updated_at = ?`,
			modelID, ruleIndex, currentCount, windowStart, time.Now(), currentCount, windowStart, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert model rate limit usage: %w", err)
		}
	} else {
		_, err := DB.Exec(
			`INSERT INTO model_rate_limit_usage (model_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(model_id, rule_index) DO UPDATE SET 
			 current_count = current_count + ?, updated_at = ?`,
			modelID, ruleIndex, currentCount, windowStart, time.Now(), currentCount, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert model rate limit usage: %w", err)
		}
	}
	return nil
}

func (r *ModelRateLimitRepo) DeleteByModel(modelID int64) error {
	_, err := DB.Exec(`DELETE FROM model_rate_limit_usage WHERE model_id = ?`, modelID)
	return err
}

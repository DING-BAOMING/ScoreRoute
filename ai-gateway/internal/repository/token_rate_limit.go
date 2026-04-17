package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type TokenRateLimitUsage struct {
	ID           int64
	TokenID      int64
	RuleIndex    int
	CurrentCount int64
	WindowStart  time.Time
}

type TokenRateLimitRepo struct{}

func NewTokenRateLimitRepo() *TokenRateLimitRepo {
	return &TokenRateLimitRepo{}
}

func (r *TokenRateLimitRepo) GetUsage(tokenID int64, ruleIndex int) (*TokenRateLimitUsage, error) {
	usage := &TokenRateLimitUsage{}
	err := DB.QueryRow(
		`SELECT id, token_id, rule_index, current_count, window_start FROM token_rate_limit_usage WHERE token_id = ? AND rule_index = ?`,
		tokenID, ruleIndex,
	).Scan(&usage.ID, &usage.TokenID, &usage.RuleIndex, &usage.CurrentCount, &usage.WindowStart)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token rate limit usage: %w", err)
	}
	return usage, nil
}

func (r *TokenRateLimitRepo) UpsertUsage(tokenID int64, ruleIndex int, currentCount int64, windowStart time.Time, resetWindow bool) error {
	if resetWindow {
		_, err := DB.Exec(
			`INSERT INTO token_rate_limit_usage (token_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(token_id, rule_index) DO UPDATE SET 
			 current_count = ?, window_start = ?, updated_at = ?`,
			tokenID, ruleIndex, currentCount, windowStart, time.Now(), currentCount, windowStart, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert token rate limit usage: %w", err)
		}
	} else {
		_, err := DB.Exec(
			`INSERT INTO token_rate_limit_usage (token_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(token_id, rule_index) DO UPDATE SET 
			 current_count = current_count + ?, updated_at = ?`,
			tokenID, ruleIndex, currentCount, windowStart, time.Now(), currentCount, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert token rate limit usage: %w", err)
		}
	}
	return nil
}

func (r *TokenRateLimitRepo) IncrementUsage(tokenID int64, ruleIndex int, increment int64, newWindowStart time.Time) error {
	_, err := DB.Exec(
		`UPDATE token_rate_limit_usage SET current_count = current_count + ?, window_start = ?, updated_at = ? WHERE token_id = ? AND rule_index = ?`,
		increment, newWindowStart, time.Now(), tokenID, ruleIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to increment token rate limit usage: %w", err)
	}
	return nil
}

func (r *TokenRateLimitRepo) DeleteByToken(tokenID int64) error {
	_, err := DB.Exec(`DELETE FROM token_rate_limit_usage WHERE token_id = ?`, tokenID)
	return err
}

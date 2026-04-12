package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type RateLimitUsage struct {
	ID           int64
	ChannelID    int64
	RuleIndex    int
	CurrentCount int64
	WindowStart  time.Time
}

type RateLimitRepo struct{}

func NewRateLimitRepo() *RateLimitRepo {
	return &RateLimitRepo{}
}

func (r *RateLimitRepo) GetUsage(channelID int64, ruleIndex int) (*RateLimitUsage, error) {
	usage := &RateLimitUsage{}
	err := DB.QueryRow(
		`SELECT id, channel_id, rule_index, current_count, window_start FROM channel_rate_limit_usage WHERE channel_id = ? AND rule_index = ?`,
		channelID, ruleIndex,
	).Scan(&usage.ID, &usage.ChannelID, &usage.RuleIndex, &usage.CurrentCount, &usage.WindowStart)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit usage: %w", err)
	}
	return usage, nil
}

func (r *RateLimitRepo) UpsertUsage(channelID int64, ruleIndex int, currentCount int64, windowStart time.Time, resetWindow bool) error {
	if resetWindow {
		_, err := DB.Exec(
			`INSERT INTO channel_rate_limit_usage (channel_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(channel_id, rule_index) DO UPDATE SET 
			 current_count = ?, window_start = ?, updated_at = ?`,
			channelID, ruleIndex, currentCount, windowStart, currentCount, windowStart, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert rate limit usage: %w", err)
		}
	} else {
		_, err := DB.Exec(
			`INSERT INTO channel_rate_limit_usage (channel_id, rule_index, current_count, window_start, updated_at) 
			 VALUES (?, ?, ?, ?, ?) 
			 ON CONFLICT(channel_id, rule_index) DO UPDATE SET 
			 current_count = current_count + ?, updated_at = ?`,
			channelID, ruleIndex, currentCount, windowStart, currentCount, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert rate limit usage: %w", err)
		}
	}
	return nil
}

func (r *RateLimitRepo) IncrementUsage(channelID int64, ruleIndex int, increment int64, newWindowStart time.Time) error {
	_, err := DB.Exec(
		`UPDATE channel_rate_limit_usage SET current_count = current_count + ?, window_start = ?, updated_at = ? WHERE channel_id = ? AND rule_index = ?`,
		increment, newWindowStart, time.Now(), channelID, ruleIndex,
	)
	if err != nil {
		return fmt.Errorf("failed to increment rate limit usage: %w", err)
	}
	return nil
}

func (r *RateLimitRepo) DeleteByChannel(channelID int64) error {
	_, err := DB.Exec(`DELETE FROM channel_rate_limit_usage WHERE channel_id = ?`, channelID)
	return err
}

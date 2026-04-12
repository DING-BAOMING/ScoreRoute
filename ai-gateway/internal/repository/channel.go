package repository

import (
	"database/sql"
	"fmt"
	"time"

	"ai-gateway/internal/model"
)

type ChannelRepo struct{}

func NewChannelRepo() *ChannelRepo {
	return &ChannelRepo{}
}

func (r *ChannelRepo) Create(req *model.ChannelRequest) (*model.Channel, error) {
	result, err := DB.Exec(
		`INSERT INTO channels (name, format, base_url, api_key, enabled, rate_limits, total_token_limit, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Format, req.BaseURL, req.APIKey, 1, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	id, _ := result.LastInsertId()
	return r.GetByID(id)
}

func (r *ChannelRepo) Update(id int64, req *model.ChannelRequest) (*model.Channel, error) {
	_, err := DB.Exec(
		`UPDATE channels SET name=?, format=?, base_url=?, api_key=?, enabled=?, rate_limits=?, total_token_limit=?, expires_at=?, updated_at=? WHERE id=?`,
		req.Name, req.Format, req.BaseURL, req.APIKey, req.Enabled, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update channel: %w", err)
	}
	return r.GetByID(id)
}

func (r *ChannelRepo) Delete(id int64) error {
	DB.Exec(`DELETE FROM models WHERE channel_id=?`, id)
	_, err := DB.Exec(`DELETE FROM channels WHERE id=?`, id)
	return err
}

func (r *ChannelRepo) GetByID(id int64) (*model.Channel, error) {
	channel := &model.Channel{}
	err := DB.QueryRow(
		`SELECT id, name, format, base_url, api_key, enabled, call_count, rate_limits, total_token_limit, expires_at, total_calls, total_tokens, created_at, updated_at FROM channels WHERE id=?`,
		id,
	).Scan(&channel.ID, &channel.Name, &channel.Format, &channel.BaseURL, &channel.APIKey, &channel.Enabled, &channel.CallCount, &channel.RateLimits, &channel.TotalTokenLimit, &channel.ExpiresAt, &channel.TotalCalls, &channel.TotalTokens, &channel.CreatedAt, &channel.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *ChannelRepo) List(page, pageSize int) ([]*model.Channel, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&total); err != nil {
		total = 0
	}

	rows, err := DB.Query(
		`SELECT id, name, format, base_url, api_key, enabled, call_count, rate_limits, total_token_limit, expires_at, total_calls, total_tokens, created_at, updated_at FROM channels ORDER BY id DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var channels []*model.Channel
	for rows.Next() {
		c := &model.Channel{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Format, &c.BaseURL, &c.APIKey, &c.Enabled, &c.CallCount, &c.RateLimits, &c.TotalTokenLimit, &c.ExpiresAt, &c.TotalCalls, &c.TotalTokens, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

func (r *ChannelRepo) GetByFormatAndType(format, modelType string) ([]*model.Channel, error) {
	rows, err := DB.Query(
		`SELECT c.id, c.name, c.format, c.base_url, c.api_key, c.enabled, c.call_count, c.rate_limits, c.total_token_limit, c.expires_at, c.total_calls, c.total_tokens, c.created_at, c.updated_at 
		 FROM channels c
		 LEFT JOIN models m ON c.id = m.channel_id AND m.enabled = 1 AND m.type = ?
		 WHERE c.format = ? AND c.enabled = 1 AND m.id IS NOT NULL
		 GROUP BY c.id`,
		modelType, format,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*model.Channel
	for rows.Next() {
		c := &model.Channel{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Format, &c.BaseURL, &c.APIKey, &c.Enabled, &c.CallCount, &c.RateLimits, &c.TotalTokenLimit, &c.ExpiresAt, &c.TotalCalls, &c.TotalTokens, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *ChannelRepo) SetEnabled(id int64, enabled int) error {
	_, err := DB.Exec(`UPDATE channels SET enabled=?, updated_at=? WHERE id=?`, enabled, time.Now(), id)
	return err
}

func (r *ChannelRepo) IncrementUsage(id int64, tokenUsed int) error {
	_, err := DB.Exec(
		`UPDATE channels SET call_count = call_count + 1, total_calls = total_calls + 1, total_tokens = total_tokens + ?, updated_at = ? WHERE id = ?`,
		tokenUsed, time.Now(), id,
	)
	return err
}

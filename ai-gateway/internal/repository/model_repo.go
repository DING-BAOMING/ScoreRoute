package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ai-gateway/internal/model"
)

type ModelRepo struct{}

func NewModelRepo() *ModelRepo {
	return &ModelRepo{}
}

func parseExpiresAt(expiresAtStr sql.NullString) *time.Time {
	if expiresAtStr.Valid && expiresAtStr.String != "" {
		t, err := time.Parse("2006-01-02 15:04:05", expiresAtStr.String)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z07:00", expiresAtStr.String)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02", expiresAtStr.String)
		}
		if err == nil {
			return &t
		}
	}
	return nil
}

func (r *ModelRepo) Create(req *model.ModelRequest) (*model.Model, error) {
	modelType := req.Type
	if modelType == "" {
		modelType = "chat"
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	result, err := DB.Exec(
		`INSERT INTO models (channel_id, name, type, enabled, rate_limits, total_token_limit, expires_at, cost_per_token, currency) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ChannelID, req.Name, modelType, 1, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt, req.CostPerToken, currency,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("该模型已存在，请勿重复添加")
		}
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	id, _ := result.LastInsertId()
	return r.GetByID(id)
}

func (r *ModelRepo) Update(id int64, req *model.ModelRequest) (*model.Model, error) {
	modelType := req.Type
	if modelType == "" {
		modelType = "chat"
	}
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	_, err := DB.Exec(
		`UPDATE models SET channel_id=?, name=?, type=?, enabled=?, rate_limits=?, total_token_limit=?, expires_at=?, cost_per_token=?, currency=? WHERE id=?`,
		req.ChannelID, req.Name, modelType, req.Enabled, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt, req.CostPerToken, currency, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}
	return r.GetByID(id)
}

func (r *ModelRepo) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM models WHERE id=?`, id)
	return err
}

func (r *ModelRepo) GetByID(id int64) (*model.Model, error) {
	model := &model.Model{}
	var expiresAtStr sql.NullString
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id WHERE m.id=?`,
		id,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAtStr.Valid && expiresAtStr.String != "" {
		t, err := time.Parse("2006-01-02 15:04:05", expiresAtStr.String)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z07:00", expiresAtStr.String)
		}
		if err != nil {
			t, err = time.Parse("2006-01-02", expiresAtStr.String)
		}
		if err == nil {
			model.ExpiresAt = &t
		}
	}
	return model, nil
}

func (r *ModelRepo) List(page, pageSize int) ([]*model.Model, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM models`).Scan(&total); err != nil {
		total = 0
	}

	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 ORDER BY m.id DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		var expiresAtStr sql.NullString
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.RateLimits, &m.TotalTokenLimit, &expiresAtStr, &m.TotalCalls, &m.TotalTokens, &m.CostPerToken, &m.Currency, &m.CreatedAt, &m.ChannelName, &m.Format); err != nil {
			continue
		}
		m.ExpiresAt = parseExpiresAt(expiresAtStr)
		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

func (r *ModelRepo) ListByChannel(channelID int64) ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id WHERE m.channel_id=? ORDER BY m.id`,
		channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		var expiresAtStr sql.NullString
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.RateLimits, &m.TotalTokenLimit, &expiresAtStr, &m.TotalCalls, &m.TotalTokens, &m.CostPerToken, &m.Currency, &m.CreatedAt, &m.ChannelName); err != nil {
			continue
		}
		m.ExpiresAt = parseExpiresAt(expiresAtStr)
		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func (r *ModelRepo) GetNextModel(channelID int64) (*model.Model, error) {
	model := &model.Model{}
	var expiresAtStr sql.NullString
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.channel_id = ? AND m.enabled = 1 AND m.auto_disabled = 0
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	model.ExpiresAt = parseExpiresAt(expiresAtStr)

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)
	DB.Exec(`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`, channelID)

	return model, nil
}

func (r *ModelRepo) GetNextModelGlobal(format, modelType string) (*model.Model, error) {
	var channelID int64
	err := DB.QueryRow(
		`SELECT c.id FROM channels c 
		 WHERE c.format = ? AND c.enabled = 1 
		 AND EXISTS (SELECT 1 FROM models m WHERE m.channel_id = c.id AND m.type = ? AND m.enabled = 1 AND m.auto_disabled = 0)
		 ORDER BY c.call_count ASC, c.id ASC LIMIT 1`,
		format, modelType,
	).Scan(&channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`, channelID)

	model := &model.Model{}
	var expiresAtStr sql.NullString
	err = DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m
		 LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.channel_id = ? AND m.type = ? AND m.enabled = 1 AND m.auto_disabled = 0 AND c.enabled = 1
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID, modelType,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	model.ExpiresAt = parseExpiresAt(expiresAtStr)

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)

	return model, nil
}

func (r *ModelRepo) GetNextModelAny() (*model.Model, error) {
	var channelID int64
	err := DB.QueryRow(
		`SELECT c.id FROM channels c 
		 WHERE c.enabled = 1 
		 AND EXISTS (SELECT 1 FROM models m WHERE m.channel_id = c.id AND m.enabled = 1 AND m.auto_disabled = 0)
		 ORDER BY c.call_count ASC, c.id ASC LIMIT 1`,
	).Scan(&channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`, channelID)

	model := &model.Model{}
	var expiresAtStr sql.NullString
	err = DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m
		 LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.channel_id = ? AND m.enabled = 1 AND m.auto_disabled = 0 AND c.enabled = 1
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	model.ExpiresAt = parseExpiresAt(expiresAtStr)

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)

	return model, nil
}

func (r *ModelRepo) SetEnabled(id int64, enabled int) error {
	_, err := DB.Exec(`UPDATE models SET enabled=? WHERE id=?`, enabled, id)
	return err
}

func (r *ModelRepo) ListEnabled() ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format, c.expires_at as channel_expires_at, c.rate_limits as channel_rate_limits
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.enabled = 1 AND m.auto_disabled = 0 AND c.enabled = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		var expiresAtStr sql.NullString
		var channelExpiresAtStr sql.NullString
		var channelRateLimitsStr sql.NullString
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.RateLimits, &m.TotalTokenLimit, &expiresAtStr, &m.TotalCalls, &m.TotalTokens, &m.CostPerToken, &m.Currency, &m.CreatedAt, &m.ChannelName, &m.Format, &channelExpiresAtStr, &channelRateLimitsStr); err != nil {
			continue
		}
		m.ExpiresAt = parseExpiresAt(expiresAtStr)
		if m.ExpiresAt == nil {
			m.ExpiresAt = parseExpiresAt(channelExpiresAtStr)
		}
		if channelRateLimitsStr.Valid {
			m.ChannelRateLimits = channelRateLimitsStr.String
		}
		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func (r *ModelRepo) GetByName(name string) (*model.Model, error) {
	model := &model.Model{}
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.name = ? AND m.enabled = 1 AND m.auto_disabled = 0 AND c.enabled = 1
		 ORDER BY m.call_count ASC LIMIT 1`,
		name,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &model.ExpiresAt, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)
	DB.Exec(`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`, model.ChannelID)

	return model, nil
}

func (r *ModelRepo) GetByNamePrefix(prefix string) ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.enabled = 1 AND m.auto_disabled = 0 AND c.enabled = 1
		 ORDER BY m.call_count ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*model.Model
	prefixLower := strings.ToLower(prefix)
	for rows.Next() {
		m := &model.Model{}
		var expiresAtStr sql.NullString
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.RateLimits, &m.TotalTokenLimit, &expiresAtStr, &m.TotalCalls, &m.TotalTokens, &m.CostPerToken, &m.Currency, &m.CreatedAt, &m.ChannelName, &m.Format); err != nil {
			continue
		}
		m.ExpiresAt = parseExpiresAt(expiresAtStr)

		modelNameLower := strings.ToLower(m.Name)
		channelNameLower := strings.ToLower(m.ChannelName)

		if strings.HasPrefix(modelNameLower, prefixLower) {
			models = append(models, m)
			continue
		}

		if strings.Contains(modelNameLower, "/") {
			parts := strings.Split(modelNameLower, "/")
			lastPart := parts[len(parts)-1]
			if strings.HasPrefix(lastPart, prefixLower) {
				models = append(models, m)
				continue
			}
		}

		if strings.HasPrefix(channelNameLower+"/"+modelNameLower, prefixLower) {
			models = append(models, m)
			continue
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func (r *ModelRepo) GetByChannelAndName(channelID int64, name string) (*model.Model, error) {
	model := &model.Model{}
	var expiresAtStr sql.NullString
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.channel_id = ? AND m.name = ?`,
		channelID, name,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	model.ExpiresAt = parseExpiresAt(expiresAtStr)
	return model, nil
}

func (r *ModelRepo) IncrementUsage(id int64, tokenUsed int) error {
	_, err := DB.Exec(
		`UPDATE models SET total_calls = total_calls + 1, total_tokens = total_tokens + ?, updated_at = ? WHERE id = ?`,
		tokenUsed, time.Now(), id,
	)
	return err
}

func (r *ModelRepo) IncrementCallCount(id int64) error {
	_, err := DB.Exec(
		`UPDATE models SET call_count = call_count + 1 WHERE id = ?`,
		id,
	)
	return err
}

func (r *ModelRepo) IncrementChannelCallCount(id int64) error {
	_, err := DB.Exec(
		`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`,
		id,
	)
	return err
}

func (r *ModelRepo) GetByChannelNameAndModel(channelName, modelName string) (*model.Model, error) {
	model := &model.Model{}
	var expiresAtStr sql.NullString
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE c.name = ? AND m.name = ?`,
		channelName, modelName,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.RateLimits, &model.TotalTokenLimit, &expiresAtStr, &model.TotalCalls, &model.TotalTokens, &model.CostPerToken, &model.Currency, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	model.ExpiresAt = parseExpiresAt(expiresAtStr)
	return model, nil
}

func (r *ModelRepo) SetAutoDisabled(id int64, reason string) error {
	_, err := DB.Exec(
		`UPDATE models SET auto_disabled=1, auto_disabled_at=?, auto_disable_reason=?, enabled=0, updated_at=? WHERE id=?`,
		time.Now(), reason, time.Now(), id,
	)
	return err
}

func (r *ModelRepo) ClearAutoDisabled(id int64) error {
	_, err := DB.Exec(
		`UPDATE models SET auto_disabled=0, auto_disabled_at=NULL, auto_disable_reason=NULL, enabled=1, updated_at=? WHERE id=?`,
		time.Now(), id,
	)
	return err
}

func (r *ModelRepo) GetAutoDisabledModels() ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.rate_limits, m.total_token_limit, m.expires_at, m.total_calls, m.total_tokens, m.cost_per_token, m.currency, m.created_at, c.name as channel_name, m.auto_disabled, m.auto_disabled_at, m.auto_disable_reason 
		FROM models m JOIN channels c ON m.channel_id = c.id WHERE m.auto_disabled=1 AND m.enabled=0`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		var expiresAtStr, autoDisabledAtStr sql.NullString
		var autoDisableReason sql.NullString
		err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.RateLimits, &m.TotalTokenLimit, &expiresAtStr, &m.TotalCalls, &m.TotalTokens, &m.CostPerToken, &m.Currency, &m.CreatedAt, &m.ChannelName, &m.AutoDisabled, &autoDisabledAtStr, &autoDisableReason)
		if err != nil {
			continue
		}
		if expiresAtStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", expiresAtStr.String); err == nil {
				m.ExpiresAt = &t
			}
		}
		if autoDisabledAtStr.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", autoDisabledAtStr.String); err == nil {
				m.AutoDisabledAt = &t
			}
		}
		models = append(models, m)
	}
	return models, nil
}

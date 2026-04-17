package repository

import (
	"database/sql"
	"fmt"

	"ai-gateway/internal/model"
)

type TokenRepo struct{}

func NewTokenRepo() *TokenRepo {
	return &TokenRepo{}
}

func (r *TokenRepo) Create(req *model.TokenRequest) (*model.Token, error) {
	result, err := DB.Exec(
		`INSERT INTO tokens (key, name, format, type, model_name, enabled, rate_limits, total_token_limit, expires_at, total_calls, total_tokens) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0)`,
		req.Key, req.Name, req.Format, req.Type, req.ModelName, req.Enabled, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	id, _ := result.LastInsertId()
	return r.GetByID(id)
}

func (r *TokenRepo) GetByID(id int64) (*model.Token, error) {
	token := &model.Token{}
	var expiresAt sql.NullTime
	err := DB.QueryRow(
		`SELECT id, key, name, format, type, model_name, enabled, rate_limits, total_token_limit, expires_at, total_calls, total_tokens, created_at FROM tokens WHERE id=?`,
		id,
	).Scan(&token.ID, &token.Key, &token.Name, &token.Format, &token.Type, &token.ModelName, &token.Enabled, &token.RateLimits, &token.TotalTokenLimit, &expiresAt, &token.TotalCalls, &token.TotalTokens, &token.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if len(token.Key) >= 4 {
		token.Key = "****" + token.Key[len(token.Key)-4:]
	} else {
		token.Key = "****"
	}
	return token, nil
}

func (r *TokenRepo) GetByKey(key string) (*model.Token, error) {
	token := &model.Token{}
	var expiresAt sql.NullTime
	err := DB.QueryRow(
		`SELECT id, key, name, format, type, model_name, enabled, rate_limits, total_token_limit, expires_at, total_calls, total_tokens, created_at FROM tokens WHERE key=?`,
		key,
	).Scan(&token.ID, &token.Key, &token.Name, &token.Format, &token.Type, &token.ModelName, &token.Enabled, &token.RateLimits, &token.TotalTokenLimit, &expiresAt, &token.TotalCalls, &token.TotalTokens, &token.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if len(token.Key) >= 4 {
		token.Key = "****" + token.Key[len(token.Key)-4:]
	} else {
		token.Key = "****"
	}
	return token, nil
}

func (r *TokenRepo) List(page, pageSize int) ([]*model.Token, int64, error) {
	offset := (page - 1) * pageSize

	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM tokens`).Scan(&total); err != nil {
		total = 0
	}

	rows, err := DB.Query(
		`SELECT id, key, name, format, type, model_name, enabled, rate_limits, total_token_limit, expires_at, total_calls, total_tokens, created_at FROM tokens ORDER BY id DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tokens []*model.Token
	for rows.Next() {
		t := &model.Token{}
		var expiresAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Key, &t.Name, &t.Format, &t.Type, &t.ModelName, &t.Enabled, &t.RateLimits, &t.TotalTokenLimit, &expiresAt, &t.TotalCalls, &t.TotalTokens, &t.CreatedAt); err != nil {
			continue
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}
		if len(t.Key) >= 4 {
			t.Key = "****" + t.Key[len(t.Key)-4:]
		} else {
			t.Key = "****"
		}
		tokens = append(tokens, t)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

func (r *TokenRepo) SetEnabled(id int64, enabled int) error {
	_, err := DB.Exec(`UPDATE tokens SET enabled=? WHERE id=?`, enabled, id)
	return err
}

func (r *TokenRepo) Update(id int64, req *model.TokenRequest) (*model.Token, error) {
	_, err := DB.Exec(
		`UPDATE tokens SET name=?, format=?, type=?, model_name=?, enabled=?, rate_limits=?, total_token_limit=?, expires_at=? WHERE id=?`,
		req.Name, req.Format, req.Type, req.ModelName, req.Enabled, req.RateLimits, req.TotalTokenLimit, req.ExpiresAt, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update token: %w", err)
	}
	return r.GetByID(id)
}

func (r *TokenRepo) Delete(id int64) error {
	_, err := DB.Exec(`DELETE FROM tokens WHERE id=?`, id)
	return err
}

func (r *TokenRepo) IncrementUsage(id int64, tokenUsed int) error {
	_, err := DB.Exec(
		`UPDATE tokens SET total_calls = total_calls + 1, total_tokens = total_tokens + ? WHERE id = ?`,
		tokenUsed, id,
	)
	return err
}
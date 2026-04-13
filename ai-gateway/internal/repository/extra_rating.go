package repository

import (
	"database/sql"
	"fmt"
	"time"

	"ai-gateway/internal/model"
)

type ExtraRatingRepo struct{}

func NewExtraRatingRepo() *ExtraRatingRepo {
	return &ExtraRatingRepo{}
}

func (r *ExtraRatingRepo) GetConfig(key string) (string, error) {
	var value string
	err := DB.QueryRow(`SELECT config_value FROM extra_rating_config WHERE config_key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	return value, nil
}

func (r *ExtraRatingRepo) SetConfig(key, value string) error {
	_, err := DB.Exec(`
		INSERT INTO extra_rating_config (config_key, config_value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(config_key) DO UPDATE SET config_value = ?, updated_at = ?`,
		key, value, time.Now(), value, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

func (r *ExtraRatingRepo) GetAllConfig() (map[string]string, error) {
	rows, err := DB.Query(`SELECT config_key, config_value FROM extra_rating_config`)
	if err != nil {
		return nil, fmt.Errorf("failed to query config: %w", err)
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		config[key] = value
	}
	return config, nil
}

func (r *ExtraRatingRepo) AddPenaltyRecord(modelKey string, penaltyScore, decayPerRequest, requestCount int, expiresAt *time.Time) error {
	_, err := DB.Exec(`DELETE FROM extra_rating_records WHERE model_key = ? AND record_type = 'penalty'`, modelKey)
	if err != nil {
		return fmt.Errorf("failed to delete existing penalty record: %w", err)
	}
	_, err = DB.Exec(`
		INSERT INTO extra_rating_records (model_key, record_type, penalty_score, current_score, decay_per_request, request_count, created_at, expires_at)
		VALUES (?, 'penalty', ?, ?, ?, ?, ?, ?)`,
		modelKey, penaltyScore, penaltyScore, decayPerRequest, requestCount, time.Now(), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to add penalty record: %w", err)
	}
	return nil
}

func (r *ExtraRatingRepo) AddRewardRecord(modelKey string, rewardScore, decayPerRequest, requestCount int, expiresAt *time.Time) error {
	_, err := DB.Exec(`
		INSERT INTO extra_rating_records (model_key, record_type, reward_score, current_score, decay_per_request, request_count, created_at, expires_at)
		VALUES (?, 'reward', ?, ?, ?, ?, ?, ?)`,
		modelKey, rewardScore, rewardScore, decayPerRequest, requestCount, time.Now(), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to add reward record: %w", err)
	}
	return nil
}

func (r *ExtraRatingRepo) GetPenaltyRecords() ([]*model.ExtraRatingRecord, error) {
	rows, err := DB.Query(`
		SELECT id, model_key, penalty_score, current_score, decay_per_request, request_count, created_at, expires_at
		FROM extra_rating_records
		WHERE record_type = 'penalty' AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query penalty records: %w", err)
	}
	defer rows.Close()

	var records []*model.ExtraRatingRecord
	for rows.Next() {
		rec := &model.ExtraRatingRecord{RecordType: "penalty"}
		var expiresAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.ModelKey, &rec.PenaltyScore, &rec.CurrentScore, &rec.DecayPerReq, &rec.RequestCount, &rec.CreatedAt, &expiresAt); err != nil {
			continue
		}
		if expiresAt.Valid {
			rec.ExpiresAt = &expiresAt.Time
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *ExtraRatingRepo) GetRewardRecords() ([]*model.ExtraRatingRecord, error) {
	rows, err := DB.Query(`
		SELECT id, model_key, reward_score, current_score, decay_per_request, request_count, created_at, expires_at
		FROM extra_rating_records
		WHERE record_type = 'reward' AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query reward records: %w", err)
	}
	defer rows.Close()

	var records []*model.ExtraRatingRecord
	for rows.Next() {
		rec := &model.ExtraRatingRecord{RecordType: "reward"}
		var expiresAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.ModelKey, &rec.RewardScore, &rec.CurrentScore, &rec.DecayPerReq, &rec.RequestCount, &rec.CreatedAt, &expiresAt); err != nil {
			continue
		}
		if expiresAt.Valid {
			rec.ExpiresAt = &expiresAt.Time
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *ExtraRatingRepo) DeleteExpiredRecords() error {
	_, err := DB.Exec(`DELETE FROM extra_rating_records WHERE expires_at IS NOT NULL AND expires_at < datetime('now')`)
	return err
}

func (r *ExtraRatingRepo) DeleteRecord(id int64) error {
	_, err := DB.Exec(`DELETE FROM extra_rating_records WHERE id = ?`, id)
	return err
}

func (r *ExtraRatingRepo) ClearAllRecords() error {
	_, err := DB.Exec(`DELETE FROM extra_rating_records`)
	return err
}

func (r *ExtraRatingRepo) IncrementPenaltyRequestCount(id int64, newRequestCount, newCurrentScore int) error {
	_, err := DB.Exec(`
		UPDATE extra_rating_records 
		SET request_count = ?, current_score = ?
		WHERE id = ?`,
		newRequestCount, newCurrentScore, id)
	return err
}

func (r *ExtraRatingRepo) UpsertRewardRecord(modelKey string, rewardScore int, expiresAt *time.Time) error {
	_, err := DB.Exec(`DELETE FROM extra_rating_records WHERE model_key = ? AND record_type = 'reward'`, modelKey)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`
		INSERT INTO extra_rating_records (model_key, record_type, reward_score, current_score, decay_per_request, request_count, created_at, expires_at)
		VALUES (?, 'reward', ?, ?, 1, 1, ?, ?)`,
		modelKey, rewardScore, rewardScore, time.Now(), expiresAt)
	return err
}

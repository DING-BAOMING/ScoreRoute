package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type ModelRatingConfigRepo struct{}

func NewModelRatingConfigRepo() *ModelRatingConfigRepo {
	return &ModelRatingConfigRepo{}
}

type ModelRatingWeights struct {
	SuccessWeight       float64 `json:"success_weight"`
	LatencyWeight      float64 `json:"latency_weight"`
	ReliabilityWeight   float64 `json:"reliability_weight"`
	UserRatingWeight    float64 `json:"user_rating_weight"`
	SampleRatingWeight  float64 `json:"sample_rating_weight"`
	CostRatingWeight    float64 `json:"cost_rating_weight"`
	TimeRatingWeight    float64 `json:"time_rating_weight"`
}

func (r *ModelRatingConfigRepo) GetAll() (*ModelRatingWeights, error) {
	weights := &ModelRatingWeights{
		SuccessWeight:      0.28,
		LatencyWeight:     0.21,
		ReliabilityWeight:  0.21,
		UserRatingWeight:   0.15,
		SampleRatingWeight: 0.15,
		CostRatingWeight:   0.0,
		TimeRatingWeight:   0.0,
	}

	rows, err := DB.Query(`SELECT config_key, config_value FROM model_rating_config`)
	if err != nil {
		return nil, fmt.Errorf("failed to query model rating config: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case "success_weight":
			weights.SuccessWeight = value
		case "latency_weight":
			weights.LatencyWeight = value
		case "reliability_weight":
			weights.ReliabilityWeight = value
		case "user_rating_weight":
			weights.UserRatingWeight = value
		case "sample_rating_weight":
			weights.SampleRatingWeight = value
		case "cost_rating_weight":
			weights.CostRatingWeight = value
		case "time_rating_weight":
			weights.TimeRatingWeight = value
		}
	}

	return weights, nil
}

func (r *ModelRatingConfigRepo) Get(key string) (float64, error) {
	var value float64
	err := DB.QueryRow(`SELECT config_value FROM model_rating_config WHERE config_key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get model rating config: %w", err)
	}
	return value, nil
}

func (r *ModelRatingConfigRepo) Set(key string, value float64) error {
	_, err := DB.Exec(
		`INSERT INTO model_rating_config (config_key, config_value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(config_key) DO UPDATE SET config_value = ?, updated_at = ?`,
		key, value, time.Now(), value, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to set model rating config: %w", err)
	}
	return nil
}

func (r *ModelRatingConfigRepo) Update(weights *ModelRatingWeights) error {
	if err := r.Set("success_weight", weights.SuccessWeight); err != nil {
		return err
	}
	if err := r.Set("latency_weight", weights.LatencyWeight); err != nil {
		return err
	}
	if err := r.Set("reliability_weight", weights.ReliabilityWeight); err != nil {
		return err
	}
	if err := r.Set("user_rating_weight", weights.UserRatingWeight); err != nil {
		return err
	}
	if err := r.Set("sample_rating_weight", weights.SampleRatingWeight); err != nil {
		return err
	}
	if err := r.Set("cost_rating_weight", weights.CostRatingWeight); err != nil {
		return err
	}
	if err := r.Set("time_rating_weight", weights.TimeRatingWeight); err != nil {
		return err
	}
	return nil
}

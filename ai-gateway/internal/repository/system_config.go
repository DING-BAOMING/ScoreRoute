package repository

import (
	"database/sql"
	"fmt"
	"time"

	"ai-gateway/internal/model"
)

type SystemConfigRepo struct{}

func NewSystemConfigRepo() *SystemConfigRepo {
	return &SystemConfigRepo{}
}

func (r *SystemConfigRepo) Get() (*model.SystemConfig, error) {
	config := &model.SystemConfig{}
	err := DB.QueryRow(
		`SELECT id, exchange_rate, currency, updated_at FROM system_config ORDER BY id DESC LIMIT 1`,
	).Scan(&config.ID, &config.ExchangeRate, &config.Currency, &config.UpdatedAt)
	if err == sql.ErrNoRows {
		config = &model.SystemConfig{
			ExchangeRate: 7.2,
			Currency:     "CNY",
			UpdatedAt:    time.Now(),
		}
		if _, err := DB.Exec(`INSERT INTO system_config (exchange_rate, currency) VALUES (?, ?)`, config.ExchangeRate, config.Currency); err != nil {
			return nil, fmt.Errorf("failed to create system config: %w", err)
		}
		return config, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get system config: %w", err)
	}
	return config, nil
}

func (r *SystemConfigRepo) Update(exchangeRate float64, currency string) error {
	_, err := DB.Exec(
		`UPDATE system_config SET exchange_rate = ?, currency = ?, updated_at = ?`,
		exchangeRate, currency, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}
	return nil
}

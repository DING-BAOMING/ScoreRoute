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
	var dispatchMode sql.NullString
	var passwordLessMode, passwordSetupDone bool
	err := DB.QueryRow(
		`SELECT id, exchange_rate, currency, dispatch_mode, password_less_mode, password_setup_done, updated_at FROM system_config ORDER BY id DESC LIMIT 1`,
	).Scan(&config.ID, &config.ExchangeRate, &config.Currency, &dispatchMode, &passwordLessMode, &passwordSetupDone, &config.UpdatedAt)
	if err == sql.ErrNoRows {
		config = &model.SystemConfig{
			ExchangeRate:      7.2,
			Currency:          "CNY",
			DispatchMode:      "polling",
			PasswordLessMode:  false,
			PasswordSetupDone: false,
			UpdatedAt:         time.Now(),
		}
		if _, err := DB.Exec(`INSERT INTO system_config (exchange_rate, currency, dispatch_mode, password_less_mode, password_setup_done) VALUES (?, ?, ?, ?, ?)`, config.ExchangeRate, config.Currency, config.DispatchMode, config.PasswordLessMode, config.PasswordSetupDone); err != nil {
			return nil, fmt.Errorf("failed to create system config: %w", err)
		}
		return config, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get system config: %w", err)
	}
	if dispatchMode.Valid {
		config.DispatchMode = dispatchMode.String
	} else {
		config.DispatchMode = "polling"
	}
	config.PasswordLessMode = passwordLessMode
	config.PasswordSetupDone = passwordSetupDone
	return config, nil
}

func (r *SystemConfigRepo) Update(exchangeRate float64, currency string, passwordLessMode bool) error {
	_, err := DB.Exec(
		`UPDATE system_config SET exchange_rate = ?, currency = ?, password_less_mode = ?, password_setup_done = 1, updated_at = ?`,
		exchangeRate, currency, passwordLessMode, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}
	return nil
}

func (r *SystemConfigRepo) UpdateDispatchMode(mode string) error {
	_, err := DB.Exec(
		`UPDATE system_config SET dispatch_mode = ?, updated_at = ?`,
		mode, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update dispatch mode: %w", err)
	}
	return nil
}

func (r *SystemConfigRepo) SetupPassword(password string) error {
	hashedPassword := hashPassword(password)
	_, err := DB.Exec(
		`UPDATE system_config SET admin_password = ?, password_setup_done = 1, password_less_mode = 0, updated_at = ?`,
		hashedPassword, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to setup password: %w", err)
	}
	return nil
}

func (r *SystemConfigRepo) EnablePasswordLessMode(enabled bool) error {
	_, err := DB.Exec(
		`UPDATE system_config SET password_less_mode = ?, password_setup_done = 1, updated_at = ?`,
		enabled, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update passwordless mode: %w", err)
	}
	return nil
}

func (r *SystemConfigRepo) GetAdminPassword() (string, error) {
	var password sql.NullString
	err := DB.QueryRow(`SELECT admin_password FROM system_config ORDER BY id DESC LIMIT 1`).Scan(&password)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get admin password: %w", err)
	}
	if password.Valid {
		return password.String, nil
	}
	return "", nil
}

func hashPassword(password string) string {
	hash := 0
	for i, c := range password {
		hash = hash*31 + int(c) + i
	}
	return fmt.Sprintf("%x", hash)
}

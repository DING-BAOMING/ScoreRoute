package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)
	DB.SetConnMaxLifetime(time.Hour)

	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			format TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'chat',
			enabled INTEGER DEFAULT 1,
			call_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (channel_id) REFERENCES channels(id),
			UNIQUE(channel_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			format TEXT NOT NULL,
			type TEXT NOT NULL,
			model_name TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS call_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_name TEXT,
			channel_name TEXT,
			model_name TEXT,
			latency_ms INTEGER DEFAULT 0,
			token_used INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_call_logs_created ON call_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_models_channel ON models(channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_key ON tokens(key)`,
		`CREATE TABLE IF NOT EXISTS user_ratings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_name TEXT NOT NULL UNIQUE,
			user_rating INTEGER DEFAULT 50,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_ratings_model ON user_ratings(model_name)`,
		`CREATE TABLE IF NOT EXISTS samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_key TEXT NOT NULL UNIQUE,
			request_content TEXT NOT NULL,
			response_content TEXT NOT NULL,
			token_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_model_key ON samples(model_key)`,
		`CREATE INDEX IF NOT EXISTS idx_samples_expires ON samples(expires_at)`,
		`CREATE TABLE IF NOT EXISTS sample_analysis_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			format TEXT NOT NULL DEFAULT 'openai',
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model_name TEXT NOT NULL DEFAULT 'gpt-4',
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sample_analysis_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_key TEXT NOT NULL,
			analysis_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			delete_time DATETIME,
			success INTEGER DEFAULT 0,
			error_message TEXT,
			score INTEGER DEFAULT 0,
			analysis_details TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sample_analysis_logs_model ON sample_analysis_logs(model_key)`,
		`CREATE INDEX IF NOT EXISTS idx_sample_analysis_logs_time ON sample_analysis_logs(analysis_time)`,
		`CREATE TABLE IF NOT EXISTS sample_ratings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_key TEXT NOT NULL UNIQUE,
			score INTEGER DEFAULT 0,
			tool_calling_score INTEGER DEFAULT 0,
			completeness_score INTEGER DEFAULT 0,
			context_understanding_score INTEGER DEFAULT 0,
			error_handling_score INTEGER DEFAULT 0,
			response_quality_score INTEGER DEFAULT 0,
			analyzed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sample_ratings_model ON sample_ratings(model_key)`,
		`CREATE TABLE IF NOT EXISTS channel_rate_limit_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			rule_index INTEGER NOT NULL,
			current_count INTEGER DEFAULT 0,
			window_start DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (channel_id) REFERENCES channels(id),
			UNIQUE(channel_id, rule_index)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_channel_rate_limit_channel ON channel_rate_limit_usage(channel_id)`,
		`CREATE TABLE IF NOT EXISTS extra_rating_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_key TEXT NOT NULL UNIQUE,
			config_value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS extra_rating_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_key TEXT NOT NULL,
			record_type TEXT NOT NULL,
			penalty_score INTEGER DEFAULT 0,
			reward_score INTEGER DEFAULT 0,
			current_score INTEGER DEFAULT 0,
			decay_per_request INTEGER DEFAULT 1,
			request_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_extra_rating_model ON extra_rating_records(model_key)`,
		`CREATE INDEX IF NOT EXISTS idx_extra_rating_type ON extra_rating_records(record_type)`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return fmt.Errorf("failed to execute: %w", err)
		}
	}

	if err := migrateTables(); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	if err := SeedDemoData(); err != nil {
		log.Printf("Warning: failed to seed demo data: %v", err)
	}

	return nil
}

func migrateTables() error {
	log.Println("Running database migration...")

	row := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name='type'")
	var channelTypeCount int
	row.Scan(&channelTypeCount)

	if channelTypeCount > 0 {
		log.Println("Found 'type' column in channels table, recreating table to remove it...")
		DB.Exec(`CREATE TABLE IF NOT EXISTS channels_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			format TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		DB.Exec(`INSERT INTO channels_new SELECT id, name, format, base_url, api_key, enabled, created_at, updated_at FROM channels`)
		DB.Exec(`DROP TABLE channels`)
		DB.Exec(`ALTER TABLE channels_new RENAME TO channels`)
		log.Println("Channels table recreated successfully")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('models') WHERE name='type'")
	var count int
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("failed to check type column: %w", err)
	}
	if count == 0 {
		log.Println("Adding 'type' column to models table...")
		if _, err := DB.Exec(`ALTER TABLE models ADD COLUMN type TEXT NOT NULL DEFAULT 'chat'`); err != nil {
			return fmt.Errorf("failed to add type column to models: %w", err)
		}
		log.Println("Migration completed successfully")
	} else {
		log.Println("Models table already has 'type' column, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name='call_count'")
	var channelCallCount int
	if err := row.Scan(&channelCallCount); err != nil {
		return fmt.Errorf("failed to check call_count column: %w", err)
	}
	if channelCallCount == 0 {
		log.Println("Adding 'call_count' column to channels table...")
		if _, err := DB.Exec(`ALTER TABLE channels ADD COLUMN call_count INTEGER DEFAULT 0`); err != nil {
			return fmt.Errorf("failed to add call_count column to channels: %w", err)
		}
		log.Println("Channel call_count column added successfully")
	} else {
		log.Println("Channels table already has 'call_count' column, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name='rate_limits'")
	var rateLimitsCount int
	if err := row.Scan(&rateLimitsCount); err != nil {
		return fmt.Errorf("failed to check rate_limits column: %w", err)
	}
	if rateLimitsCount == 0 {
		log.Println("Adding rate limiting columns to channels table...")
		DB.Exec(`ALTER TABLE channels ADD COLUMN rate_limits TEXT DEFAULT '[]'`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN total_token_limit INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN expires_at DATETIME`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN total_calls INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN total_tokens INTEGER DEFAULT 0`)
		log.Println("Rate limiting columns added successfully")
	} else {
		log.Println("Channels table already has rate limiting columns, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('models') WHERE name='rate_limits'")
	var modelRateLimitsCount int
	if err := row.Scan(&modelRateLimitsCount); err != nil {
		return fmt.Errorf("failed to check rate_limits column in models: %w", err)
	}
	if modelRateLimitsCount == 0 {
		log.Println("Adding rate limiting columns to models table...")
		DB.Exec(`ALTER TABLE models ADD COLUMN rate_limits TEXT DEFAULT '[]'`)
		DB.Exec(`ALTER TABLE models ADD COLUMN total_token_limit INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE models ADD COLUMN expires_at DATETIME`)
		DB.Exec(`ALTER TABLE models ADD COLUMN total_calls INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE models ADD COLUMN total_tokens INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE models ADD COLUMN cost_per_token REAL DEFAULT 0`)
		DB.Exec(`ALTER TABLE models ADD COLUMN currency TEXT DEFAULT 'CNY'`)
		log.Println("Model rate limiting columns added successfully")
	} else {
		log.Println("Models table already has rate limiting columns, skipping")
	}

	var modelUpdatedAtCount int
	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('models') WHERE name='updated_at'")
	if err := row.Scan(&modelUpdatedAtCount); err != nil {
		return fmt.Errorf("failed to check updated_at column in models: %w", err)
	}
	if modelUpdatedAtCount == 0 {
		log.Println("Adding updated_at column to models table...")
		DB.Exec(`ALTER TABLE models ADD COLUMN updated_at DATETIME`)
		log.Println("Model updated_at column added successfully")
	} else {
		log.Println("Models table already has updated_at column, skipping")
	}

	var tableCount int
	row = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='model_rate_limit_usage'")
	if err := row.Scan(&tableCount); err != nil {
		return fmt.Errorf("failed to check model_rate_limit_usage table: %w", err)
	}
	if tableCount == 0 {
		log.Println("Creating model_rate_limit_usage table...")
		DB.Exec(`CREATE TABLE model_rate_limit_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_id INTEGER NOT NULL,
			rule_index INTEGER NOT NULL,
			current_count INTEGER DEFAULT 0,
			window_start DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (model_id) REFERENCES models(id),
			UNIQUE(model_id, rule_index)
		)`)
		DB.Exec(`CREATE INDEX idx_model_rate_limit_model ON model_rate_limit_usage(model_id)`)
		log.Println("Model rate limit usage table created successfully")
	} else {
		log.Println("Model rate limit usage table already exists, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='system_config'")
	if err := row.Scan(&tableCount); err != nil {
		return fmt.Errorf("failed to check system_config table: %w", err)
	}
	if tableCount == 0 {
		log.Println("Creating system_config table...")
		DB.Exec(`CREATE TABLE system_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			exchange_rate REAL DEFAULT 7.2,
			currency TEXT DEFAULT 'CNY',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		DB.Exec(`INSERT INTO system_config (exchange_rate, currency) VALUES (7.2, 'CNY')`)
		log.Println("System config table created successfully")
	} else {
		log.Println("System config table already exists, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='model_rating_config'")
	if err := row.Scan(&tableCount); err != nil {
		return fmt.Errorf("failed to check model_rating_config table: %w", err)
	}
	if tableCount == 0 {
		log.Println("Creating model_rating_config table...")
		DB.Exec(`CREATE TABLE model_rating_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_key TEXT NOT NULL UNIQUE,
			config_value REAL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('success_weight', 0.28)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('latency_weight', 0.21)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('reliability_weight', 0.21)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('user_rating_weight', 0.15)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('sample_rating_weight', 0.15)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('cost_rating_weight', 0.0)`)
		DB.Exec(`INSERT INTO model_rating_config (config_key, config_value) VALUES ('time_rating_weight', 0.0)`)
		log.Println("Model rating config table created successfully")
	} else {
		log.Println("Model rating config table already exists, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_extra_rating_model_type'")
	if err := row.Scan(&tableCount); err != nil {
		return fmt.Errorf("failed to check unique index: %w", err)
	}
	if tableCount > 0 {
		log.Println("Dropping unique index on extra_rating_records (not needed)...")
		DB.Exec(`DROP INDEX IF EXISTS idx_extra_rating_model_type`)
		log.Println("Unique index dropped successfully")
	} else {
		log.Println("Unique index does not exist, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('system_config') WHERE name='dispatch_mode'")
	var dispatchModeCount int
	if err := row.Scan(&dispatchModeCount); err != nil {
		return fmt.Errorf("failed to check dispatch_mode column: %w", err)
	}
	if dispatchModeCount == 0 {
		log.Println("Adding dispatch_mode column to system_config table...")
		if _, err := DB.Exec(`ALTER TABLE system_config ADD COLUMN dispatch_mode TEXT DEFAULT 'polling'`); err != nil {
			return fmt.Errorf("failed to add dispatch_mode column to system_config: %w", err)
		}
		log.Println("dispatch_mode column added successfully")
	} else {
		log.Println("system_config table already has dispatch_mode column, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tokens') WHERE name='rate_limits'")
	var tokenRateLimitsCount int
	if err := row.Scan(&tokenRateLimitsCount); err != nil {
		return fmt.Errorf("failed to check rate_limits column in tokens: %w", err)
	}
	if tokenRateLimitsCount == 0 {
		log.Println("Adding rate limiting columns to tokens table...")
		DB.Exec(`ALTER TABLE tokens ADD COLUMN rate_limits TEXT DEFAULT '[]'`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN total_token_limit INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN expires_at DATETIME`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN total_calls INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN total_tokens INTEGER DEFAULT 0`)
		log.Println("Token rate limiting columns added successfully")
	} else {
		log.Println("Tokens table already has rate limiting columns, skipping")
	}

	var tokenRateLimitTableCount int
	row = DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='token_rate_limit_usage'")
	if err := row.Scan(&tokenRateLimitTableCount); err != nil {
		return fmt.Errorf("failed to check token_rate_limit_usage table: %w", err)
	}
	if tokenRateLimitTableCount == 0 {
		log.Println("Creating token_rate_limit_usage table...")
		DB.Exec(`CREATE TABLE token_rate_limit_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_id INTEGER NOT NULL,
			rule_index INTEGER NOT NULL,
			current_count INTEGER DEFAULT 0,
			window_start DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (token_id) REFERENCES tokens(id),
			UNIQUE(token_id, rule_index)
		)`)
		DB.Exec(`CREATE INDEX idx_token_rate_limit_token ON token_rate_limit_usage(token_id)`)
		log.Println("Token rate limit usage table created successfully")
	} else {
		log.Println("Token rate limit usage table already exists, skipping")
	}

	// Migration: Add auto_disabled columns to models, tokens, and channels
	// This is needed because the code references auto_disabled column but it was never created
	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('models') WHERE name='auto_disabled'")
	var modelsAutoDisabledCount int
	if err := row.Scan(&modelsAutoDisabledCount); err != nil {
		log.Printf("Warning: failed to check auto_disabled column in models: %v", err)
	}
	if modelsAutoDisabledCount == 0 {
		log.Println("Adding auto_disabled columns to models table...")
		DB.Exec(`ALTER TABLE models ADD COLUMN auto_disabled INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE models ADD COLUMN auto_disabled_at DATETIME`)
		DB.Exec(`ALTER TABLE models ADD COLUMN auto_disable_reason TEXT`)
		log.Println("Models auto_disabled columns added")
	} else {
		log.Println("Models table already has auto_disabled column, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tokens') WHERE name='auto_disabled'")
	var tokensAutoDisabledCount int
	if err := row.Scan(&tokensAutoDisabledCount); err != nil {
		log.Printf("Warning: failed to check auto_disabled column in tokens: %v", err)
	}
	if tokensAutoDisabledCount == 0 {
		log.Println("Adding auto_disabled columns to tokens table...")
		DB.Exec(`ALTER TABLE tokens ADD COLUMN auto_disabled INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN auto_disabled_at DATETIME`)
		DB.Exec(`ALTER TABLE tokens ADD COLUMN auto_disable_reason TEXT`)
		log.Println("Tokens auto_disabled columns added")
	} else {
		log.Println("Tokens table already has auto_disabled column, skipping")
	}

	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('channels') WHERE name='auto_disabled'")
	var channelsAutoDisabledCount int
	if err := row.Scan(&channelsAutoDisabledCount); err != nil {
		log.Printf("Warning: failed to check auto_disabled column in channels: %v", err)
	}
	if channelsAutoDisabledCount == 0 {
		log.Println("Adding auto_disabled columns to channels table...")
		DB.Exec(`ALTER TABLE channels ADD COLUMN auto_disabled INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN auto_disabled_at DATETIME`)
		DB.Exec(`ALTER TABLE channels ADD COLUMN auto_disable_reason TEXT`)
		log.Println("Channels auto_disabled columns added")
	} else {
		log.Println("Channels table already has auto_disabled column, skipping")
	}

	// Migration: Add password columns to system_config
	var passwordLessModeCount int
	row = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('system_config') WHERE name='password_less_mode'")
	if err := row.Scan(&passwordLessModeCount); err != nil {
		log.Printf("Warning: failed to check password_less_mode column: %v", err)
	}
	if passwordLessModeCount == 0 {
		log.Println("Adding password columns to system_config table...")
		DB.Exec(`ALTER TABLE system_config ADD COLUMN password_less_mode INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE system_config ADD COLUMN password_setup_done INTEGER DEFAULT 0`)
		DB.Exec(`ALTER TABLE system_config ADD COLUMN admin_password TEXT`)
		log.Println("Password columns added successfully")
	} else {
		log.Println("system_config table already has password columns, skipping")
	}

	return nil
}

func SeedDemoData() error {
	var channelCount int
	row := DB.QueryRow("SELECT COUNT(*) FROM channels")
	if err := row.Scan(&channelCount); err != nil {
		return fmt.Errorf("failed to check channel count: %w", err)
	}

	if channelCount > 0 {
		log.Println("Channels already exist, skipping seed data")
		return nil
	}

	log.Println("Seeding demo MiniMax channel and models...")

	_, err := DB.Exec(`
		INSERT INTO channels (name, format, base_url, api_key, enabled, rate_limits, total_token_limit)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Demo-MiniMax",
		"openai",
		"https://api.minimax.chat/v1",
		"YOUR_MINIMAX_API_KEY",
		1,
		"[]",
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to insert demo channel: %w", err)
	}

	var channelID int64
	row = DB.QueryRow("SELECT id FROM channels WHERE name = 'Demo-MiniMax'")
	if err := row.Scan(&channelID); err != nil {
		return fmt.Errorf("failed to get demo channel ID: %w", err)
	}

	models := []struct {
		name     string
		modelType string
	}{
		{"MiniMax-M2.7", "chat"},
		{"MiniMax-Text-01", "chat"},
		{"abab6-chat", "chat"},
	}

	for _, m := range models {
		_, err := DB.Exec(`
			INSERT INTO models (channel_id, name, type, enabled, rate_limits, total_token_limit, cost_per_token, currency)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			channelID,
			m.name,
			m.modelType,
			1,
			"[]",
			0,
			0,
			"CNY",
		)
		if err != nil {
			log.Printf("Warning: failed to insert demo model %s: %v", m.name, err)
		}
	}

	log.Println("Demo seed data inserted successfully")
	return nil
}

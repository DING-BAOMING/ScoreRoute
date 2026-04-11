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
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return fmt.Errorf("failed to execute: %w", err)
		}
	}

	if err := migrateTables(); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
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

	return nil
}

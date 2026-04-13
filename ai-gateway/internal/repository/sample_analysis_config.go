package repository

import (
	"ai-gateway/internal/model"
	"database/sql"
)

type SampleAnalysisConfigRepo struct{}

func NewSampleAnalysisConfigRepo() *SampleAnalysisConfigRepo {
	return &SampleAnalysisConfigRepo{}
}

func (r *SampleAnalysisConfigRepo) Get() (*model.SampleAnalysisConfig, error) {
	cfg := &model.SampleAnalysisConfig{}
	err := DB.QueryRow(`
		SELECT id, format, base_url, api_key, model_name, enabled, created_at, updated_at 
		FROM sample_analysis_config LIMIT 1
	`).Scan(&cfg.ID, &cfg.Format, &cfg.BaseURL, &cfg.APIKey, &cfg.ModelName, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(cfg.APIKey) >= 4 {
		cfg.MaskedAPIKey = "****" + cfg.APIKey[len(cfg.APIKey)-4:]
	} else {
		cfg.MaskedAPIKey = "****"
	}
	return cfg, nil
}

func (r *SampleAnalysisConfigRepo) Upsert(req *model.SampleAnalysisConfigRequest) error {
	existing, err := r.Get()
	if err != nil {
		return err
	}

	if existing == nil {
		_, err = DB.Exec(`
			INSERT INTO sample_analysis_config (format, base_url, api_key, model_name, enabled)
			VALUES (?, ?, ?, ?, ?)
		`, req.Format, req.BaseURL, req.APIKey, req.ModelName, req.Enabled)
	} else {
		_, err = DB.Exec(`
			UPDATE sample_analysis_config 
			SET format = ?, base_url = ?, api_key = ?, model_name = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, req.Format, req.BaseURL, req.APIKey, req.ModelName, req.Enabled, existing.ID)
	}
	return err
}

func (r *SampleAnalysisConfigRepo) GetEnabled() (*model.SampleAnalysisConfig, error) {
	cfg := &model.SampleAnalysisConfig{}
	err := DB.QueryRow(`
		SELECT id, format, base_url, api_key, model_name, enabled, created_at, updated_at 
		FROM sample_analysis_config WHERE enabled = 1 LIMIT 1
	`).Scan(&cfg.ID, &cfg.Format, &cfg.BaseURL, &cfg.APIKey, &cfg.ModelName, &cfg.Enabled, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(cfg.APIKey) >= 4 {
		cfg.MaskedAPIKey = "****" + cfg.APIKey[len(cfg.APIKey)-4:]
	} else {
		cfg.MaskedAPIKey = "****"
	}
	return cfg, nil
}

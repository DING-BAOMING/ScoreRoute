package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"ai-gateway/internal/model"
)

type ModelRepo struct{}

func NewModelRepo() *ModelRepo {
	return &ModelRepo{}
}

func (r *ModelRepo) Create(req *model.ModelRequest) (*model.Model, error) {
	modelType := req.Type
	if modelType == "" {
		modelType = "chat"
	}
	result, err := DB.Exec(
		`INSERT INTO models (channel_id, name, type, enabled) VALUES (?, ?, ?, ?)`,
		req.ChannelID, req.Name, modelType, 1,
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
	_, err := DB.Exec(
		`UPDATE models SET channel_id=?, name=?, type=?, enabled=? WHERE id=?`,
		req.ChannelID, req.Name, modelType, req.Enabled, id,
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
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name, c.format as channel_format
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id WHERE m.id=?`,
		id,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName, &model.Format)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
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
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name, c.format as channel_format
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
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.CreatedAt, &m.ChannelName, &m.Format); err != nil {
			continue
		}
		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

func (r *ModelRepo) ListByChannel(channelID int64) ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
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
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.CreatedAt, &m.ChannelName); err != nil {
			continue
		}
		models = append(models, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func (r *ModelRepo) GetNextModel(channelID int64) (*model.Model, error) {
	model := &model.Model{}
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id 
		 WHERE m.channel_id = ? AND m.enabled = 1
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)
	DB.Exec(`UPDATE channels SET call_count = call_count + 1 WHERE id = ?`, channelID)

	return model, nil
}

func (r *ModelRepo) GetNextModelGlobal(format, modelType string) (*model.Model, error) {
	var channelID int64
	err := DB.QueryRow(
		`SELECT c.id FROM channels c 
		 WHERE c.format = ? AND c.enabled = 1 
		 AND EXISTS (SELECT 1 FROM models m WHERE m.channel_id = c.id AND m.type = ? AND m.enabled = 1)
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
	err = DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m 
		 LEFT JOIN channels c ON m.channel_id = c.id 
		 WHERE m.channel_id = ? AND m.type = ? AND m.enabled = 1 AND c.enabled = 1
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID, modelType,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)

	return model, nil
}

func (r *ModelRepo) GetNextModelAny() (*model.Model, error) {
	var channelID int64
	err := DB.QueryRow(
		`SELECT c.id FROM channels c 
		 WHERE c.enabled = 1 
		 AND EXISTS (SELECT 1 FROM models m WHERE m.channel_id = c.id AND m.enabled = 1)
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
	err = DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m 
		 LEFT JOIN channels c ON m.channel_id = c.id 
		 WHERE m.channel_id = ? AND m.enabled = 1 AND c.enabled = 1
		 ORDER BY m.call_count ASC, m.id ASC LIMIT 1`,
		channelID,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	DB.Exec(`UPDATE models SET call_count = call_count + 1 WHERE id = ?`, model.ID)

	return model, nil
}

func (r *ModelRepo) SetEnabled(id int64, enabled int) error {
	_, err := DB.Exec(`UPDATE models SET enabled=? WHERE id=?`, enabled, id)
	return err
}

func (r *ModelRepo) ListEnabled() ([]*model.Model, error) {
	rows, err := DB.Query(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.enabled = 1 AND c.enabled = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Name, &m.Type, &m.Enabled, &m.CallCount, &m.CreatedAt, &m.ChannelName); err != nil {
			continue
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
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id
		 WHERE m.name = ? AND m.enabled = 1 AND c.enabled = 1
		 ORDER BY m.call_count ASC LIMIT 1`,
		name,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName)
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

func (r *ModelRepo) GetByChannelAndName(channelID int64, name string) (*model.Model, error) {
	model := &model.Model{}
	err := DB.QueryRow(
		`SELECT m.id, m.channel_id, m.name, m.type, m.enabled, m.call_count, m.created_at, c.name as channel_name
		 FROM models m LEFT JOIN channels c ON m.channel_id = c.id 
		 WHERE m.channel_id = ? AND m.name = ?`,
		channelID, name,
	).Scan(&model.ID, &model.ChannelID, &model.Name, &model.Type, &model.Enabled, &model.CallCount, &model.CreatedAt, &model.ChannelName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model, nil
}

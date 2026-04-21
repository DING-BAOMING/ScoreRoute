package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type ModelService struct {
	repo        *repository.ModelRepo
	channelRepo *repository.ChannelRepo
	client      *http.Client
}

func NewModelService() *ModelService {
	return &ModelService{
		repo:        repository.NewModelRepo(),
		channelRepo: repository.NewChannelRepo(),
		client:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *ModelService) Create(req *model.ModelRequest) (*model.Model, error) {
	channel, err := s.channelRepo.GetByID(req.ChannelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, nil
	}

	return s.repo.Create(req)
}

func (s *ModelService) Update(id int64, req *model.ModelRequest) (*model.Model, error) {
	return s.repo.Update(id, req)
}

func (s *ModelService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ModelService) GetByID(id int64) (*model.Model, error) {
	return s.repo.GetByID(id)
}

func (s *ModelService) List(page, pageSize int) ([]*model.Model, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize)
}

func (s *ModelService) ListByChannel(channelID int64) ([]*model.Model, error) {
	return s.repo.ListByChannel(channelID)
}

func (s *ModelService) SetEnabled(id int64, enabled int) error {
	return s.repo.SetEnabled(id, enabled)
}

func (s *ModelService) ListEnabled() ([]*model.Model, error) {
	return s.repo.ListEnabled()
}

func (s *ModelService) GetByName(name string) (*model.Model, error) {
	return s.repo.GetByName(name)
}

func (s *ModelService) GetByNamePrefix(prefix string) ([]*model.Model, error) {
	return s.repo.GetByNamePrefix(prefix)
}

func (s *ModelService) GetNextModel(channelID int64) (*model.Model, error) {
	return s.repo.GetNextModel(channelID)
}

func (s *ModelService) GetNextModelGlobal(format, modelType string) (*model.Model, error) {
	return s.repo.GetNextModelGlobal(format, modelType)
}

func (s *ModelService) GetNextModelAny() (*model.Model, error) {
	return s.repo.GetNextModelAny()
}

func (s *ModelService) GetByChannelAndName(channelID int64, name string) (*model.Model, error) {
	return s.repo.GetByChannelAndName(channelID, name)
}

func (s *ModelService) BatchCreate(channelID int64, modelNames []string, modelType string) ([]string, error) {
	createdKeys := []string{}
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil || channel == nil {
		return createdKeys, nil
	}

	for _, name := range modelNames {
		existing, _ := s.repo.GetByChannelAndName(channelID, name)
		if existing == nil {
			modelReq := &model.ModelRequest{
				ChannelID: channelID,
				Name:      name,
				Type:      modelType,
				Enabled:   1,
			}
			created, err := s.repo.Create(modelReq)
			if err == nil && created != nil {
				modelKey := NormalizeModelKey(channel.Name, created.Format, created.Type, created.Name)
				createdKeys = append(createdKeys, modelKey)
			}
		}
	}
	return createdKeys, nil
}

func (s *ModelService) TestModel(modelID int64) (map[string]interface{}, error) {
	modelItem, err := s.repo.GetByID(modelID)
	if err != nil || modelItem == nil {
		return nil, fmt.Errorf("model not found")
	}

	channel, err := s.channelRepo.GetByID(modelItem.ChannelID)
	if err != nil || channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	startTime := time.Now()

	reqBody := map[string]interface{}{
		"model": modelItem.Name,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 10,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return map[string]interface{}{
			"success":    false,
			"error":      fmt.Sprintf("failed to marshal request: %v", err),
			"latency_ms": time.Since(startTime).Milliseconds(),
		}, nil
	}

	url := fmt.Sprintf("%s/chat/completions", channel.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return map[string]interface{}{
			"success":    false,
			"error":      fmt.Sprintf("failed to create request: %v", err),
			"latency_ms": time.Since(startTime).Milliseconds(),
		}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"success":    false,
			"error":      err.Error(),
			"latency_ms": time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success":     false,
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("failed to read response: %v", err),
			"latency_ms":  time.Since(startTime).Milliseconds(),
		}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return map[string]interface{}{
			"success":     false,
			"status_code": resp.StatusCode,
			"error":       fmt.Sprintf("failed to parse response: %v", err),
			"latency_ms":  time.Since(startTime).Milliseconds(),
		}, nil
	}

	return map[string]interface{}{
		"success":     resp.StatusCode == 200,
		"status_code": resp.StatusCode,
		"latency_ms":  time.Since(startTime).Milliseconds(),
		"response":    result,
	}, nil
}

func (s *ModelService) GetAutoDisabledModels() ([]*model.Model, error) {
	return s.repo.GetAutoDisabledModels()
}

package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type ChannelService struct {
	repo   *repository.ChannelRepo
	client *http.Client
}

func NewChannelService() *ChannelService {
	return &ChannelService{
		repo:   repository.NewChannelRepo(),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *ChannelService) Create(req *model.ChannelRequest) (*model.Channel, error) {
	return s.repo.Create(req)
}

func (s *ChannelService) Update(id int64, req *model.ChannelRequest) (*model.Channel, error) {
	return s.repo.Update(id, req)
}

func (s *ChannelService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ChannelService) GetByID(id int64) (*model.Channel, error) {
	return s.repo.GetByID(id)
}

func (s *ChannelService) List(page, pageSize int) ([]*model.Channel, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize)
}

func (s *ChannelService) SetEnabled(id int64, enabled int) error {
	return s.repo.SetEnabled(id, enabled)
}

func (s *ChannelService) GetByFormatAndType(format, modelType string) ([]*model.Channel, error) {
	return s.repo.GetByFormatAndType(format, modelType)
}

func (s *ChannelService) TestCredentials(baseURL, apiKey string) (bool, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/chat/completions"

	testModel := "gpt-3.5-turbo"
	if strings.Contains(baseURL, "minimax") {
		testModel = "MiniMax-M2.5"
	} else if strings.Contains(baseURL, "zhipu") || strings.Contains(baseURL, "bigmodel") {
		testModel = "glm-4"
	} else if strings.Contains(baseURL, "nvapi") || strings.Contains(baseURL, "nvidia") {
		testModel = "meta/llama-3.1-405b-instruct"
	}

	testBody := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"test"}],"max_tokens":1}`, testModel)
	req, err := http.NewRequest("POST", url, strings.NewReader(testBody))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return true, nil
}

func (s *ChannelService) FetchAvailableModels(channelID int64) ([]string, error) {
	channel, err := s.repo.GetByID(channelID)
	if err != nil || channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	url := strings.TrimSuffix(channel.BaseURL, "/") + "/models"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+channel.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("连接失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}

	return models, nil
}

func (s *ChannelService) ListAll() ([]*model.Channel, error) {
	return s.repo.ListAll()
}

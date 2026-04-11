package service

import (
	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"time"
)

type SampleService struct {
	repo *repository.SampleRepo
}

func NewSampleService() *SampleService {
	return &SampleService{
		repo: repository.NewSampleRepo(),
	}
}

func (s *SampleService) CreateSample(modelKey, requestContent, responseContent string, tokenCount int) error {
	return s.repo.SaveSample(modelKey, requestContent, responseContent, tokenCount)
}

func (s *SampleService) GetByModelKey(modelKey string) (*model.Sample, error) {
	return s.repo.GetByModelKey(modelKey)
}

func (s *SampleService) ListSamples() ([]*model.Sample, error) {
	return s.repo.List()
}

func (s *SampleService) DeleteSample(id int64) error {
	return s.repo.Delete(id)
}

func (s *SampleService) CleanupExpired() (int64, error) {
	return s.repo.DeleteExpired()
}

func (s *SampleService) GetStats() (map[string]interface{}, error) {
	return s.repo.GetStats()
}

func (s *SampleService) CleanupOldSamples(days int) (int64, error) {
	return s.repo.CleanupOlderThan(days)
}

func GetRemainingDays(expiresAt time.Time) int {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}
	days := int(remaining.Hours() / 24)
	if days < 1 {
		return 1
	}
	return days
}

func GetRemainingMinutes(expiresAt time.Time) int {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}
	minutes := int(remaining.Minutes())
	if minutes < 1 {
		return 1
	}
	return minutes
}

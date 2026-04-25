package service

import (
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type LogService struct {
	repo *repository.LogRepo
}

func NewLogService() *LogService {
	return &LogService{
		repo: repository.NewLogRepo(),
	}
}

func (s *LogService) List(page, pageSize int, startTime, endTime *time.Time, model, status, tokenId, channelId string) ([]*model.CallLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, startTime, endTime, model, status, tokenId, channelId)
}

func (s *LogService) GetStats() (map[string]interface{}, error) {
	return s.repo.GetStats()
}

func (s *LogService) GetDashboard() (*model.PageResult, error) {
	return s.repo.GetTotalStats()
}

func (s *LogService) Cleanup(days int) error {
	return s.repo.CleanupOlderThan(days)
}

func (s *LogService) GetModelStats() ([]map[string]interface{}, error) {
	return s.repo.GetModelStats()
}

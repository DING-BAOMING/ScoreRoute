package service

import (
	"ai-gateway/internal/model"
)

func (s *SampleAnalysisService) GetLogs(limit int) ([]*model.SampleAnalysisLog, error) {
	return s.logRepo.List(limit)
}

func (s *SampleAnalysisService) GetLogStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	logs, err := s.logRepo.List(100)
	if err != nil {
		return stats, err
	}

	var total, success, failed int
	var totalScore int
	for _, log := range logs {
		total++
		if log.Success == 1 {
			success++
			totalScore += log.Score
		} else {
			failed++
		}
	}

	stats["total"] = total
	stats["success"] = success
	stats["failed"] = failed
	if success > 0 {
		stats["avg_score"] = totalScore / success
	} else {
		stats["avg_score"] = 0
	}

	return stats, nil
}

func (s *SampleAnalysisService) GetRatings() ([]*model.SampleRating, error) {
	return s.ratingRepo.List()
}

func (s *SampleAnalysisService) GetRatingsMap() (map[string]*model.SampleRating, error) {
	return s.ratingRepo.GetAllAsMap()
}

func (s *SampleAnalysisService) UpdateRating(modelKey string, score int) error {
	rating := &model.SampleRating{
		ModelKey: modelKey,
		Score:    score,
	}
	return s.ratingRepo.Upsert(rating)
}

func (s *SampleAnalysisService) CleanupExpiredRatings() error {
	_, err := s.ratingRepo.DeleteExpired()
	return err
}

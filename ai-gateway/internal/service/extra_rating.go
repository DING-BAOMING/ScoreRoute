package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type ExtraRatingService struct {
	repo      *repository.ExtraRatingRepo
	modelRepo *repository.ModelRepo
}

func NewExtraRatingService() *ExtraRatingService {
	return &ExtraRatingService{
		repo:      repository.NewExtraRatingRepo(),
		modelRepo: repository.NewModelRepo(),
	}
}

type ExtraRatingConfig struct {
	PunishmentRounds int // Number of rounds punishment lasts
	PunishmentScore  int // Score deducted per call
	RewardHours      int // Hours reward lasts after call
	RewardScore      int // Score added per call
}

func (s *ExtraRatingService) GetConfig() (*ExtraRatingConfig, error) {
	configMap, err := s.repo.GetAllConfig()
	if err != nil {
		return nil, err
	}

	config := &ExtraRatingConfig{
		PunishmentRounds: 5,
		PunishmentScore:  5,
		RewardHours:      24,
		RewardScore:      5,
	}

	if val, ok := configMap["punishment_rounds"]; ok {
		fmt.Sscanf(val, "%d", &config.PunishmentRounds)
	}
	if val, ok := configMap["punishment_score"]; ok {
		fmt.Sscanf(val, "%d", &config.PunishmentScore)
	}
	if val, ok := configMap["reward_hours"]; ok {
		fmt.Sscanf(val, "%d", &config.RewardHours)
	}
	if val, ok := configMap["reward_score"]; ok {
		fmt.Sscanf(val, "%d", &config.RewardScore)
	}

	return config, nil
}

func (s *ExtraRatingService) SetConfig(config *ExtraRatingConfig) error {
	if err := s.repo.SetConfig("punishment_rounds", fmt.Sprintf("%d", config.PunishmentRounds)); err != nil {
		return err
	}
	if err := s.repo.SetConfig("punishment_score", fmt.Sprintf("%d", config.PunishmentScore)); err != nil {
		return err
	}
	if err := s.repo.SetConfig("reward_hours", fmt.Sprintf("%d", config.RewardHours)); err != nil {
		return err
	}
	if err := s.repo.SetConfig("reward_score", fmt.Sprintf("%d", config.RewardScore)); err != nil {
		return err
	}
	return nil
}

func (s *ExtraRatingService) GetRecords() (*model.ExtraRatingResponse, error) {
	penaltyRecords, err := s.repo.GetPenaltyRecords()
	if err != nil {
		return nil, err
	}

	rewardRecords, err := s.repo.GetRewardRecords()
	if err != nil {
		return nil, err
	}

	response := &model.ExtraRatingResponse{
		PenaltyRecords: make([]model.ExtraRatingRecord, 0),
		RewardRecords:  make([]model.ExtraRatingRecord, 0),
	}

	for _, p := range penaltyRecords {
		response.PenaltyRecords = append(response.PenaltyRecords, *p)
	}
	for _, r := range rewardRecords {
		response.RewardRecords = append(response.RewardRecords, *r)
	}

	return response, nil
}

func (s *ExtraRatingService) ClearRecords() error {
	return s.repo.ClearAllRecords()
}

func (s *ExtraRatingService) DeleteRecord(id int64) error {
	return s.repo.DeleteRecord(id)
}

func (s *ExtraRatingService) GetPenaltyRecords() ([]*model.ExtraRatingRecord, error) {
	return s.repo.GetPenaltyRecords()
}

func (s *ExtraRatingService) GetRewardRecords() ([]*model.ExtraRatingRecord, error) {
	return s.repo.GetRewardRecords()
}

func (s *ExtraRatingService) ApplyPenaltyAndReward(modelKey string) error {
	config, err := s.GetConfig()
	if err != nil {
		return err
	}

	penaltyRecords, err := s.repo.GetPenaltyRecords()
	if err != nil {
		return err
	}

	decayPerRequest := 1
	if config.PunishmentRounds > 0 {
		decayPerRequest = config.PunishmentScore / config.PunishmentRounds
	}

	penaltyScore := -5
	if config.PunishmentScore > 0 {
		penaltyScore = -config.PunishmentScore
	}

	for _, p := range penaltyRecords {
		newScore := p.CurrentScore + decayPerRequest
		if newScore >= 0 {
			s.repo.DeleteRecord(p.ID)
		} else {
			s.repo.UpdatePenaltyScore(p.ID, newScore)
		}
	}

	if err := s.repo.AddPenaltyRecord(modelKey, penaltyScore, decayPerRequest, 0, nil); err != nil {
		return fmt.Errorf("failed to add penalty record: %w", err)
	}

	return nil
}

func (s *ExtraRatingService) ApplyPenaltyAndRewardContext(ctx context.Context, modelKey string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ApplyPenaltyAndReward(modelKey)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ExtraRatingService) ApplyNewModelReward(modelKey string) error {
	config, err := s.GetConfig()
	if err != nil {
		return err
	}

	rewardExpiresAt := time.Now().Add(time.Duration(config.RewardHours) * time.Hour)
	if err := s.repo.UpsertRewardRecord(modelKey, config.RewardScore, &rewardExpiresAt); err != nil {
		return err
	}

	return nil
}

func (s *ExtraRatingService) GetModelExtraScore(modelKey string) (int, int, error) {
	penaltyRecords, err := s.repo.GetPenaltyRecords()
	if err != nil {
		return 0, 0, err
	}

	rewardRecords, err := s.repo.GetRewardRecords()
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	totalPenalty := 0
	for _, p := range penaltyRecords {
		if p.ModelKey == modelKey {
			totalPenalty += p.CurrentScore
		}
	}

	totalReward := 0
	for _, r := range rewardRecords {
		if r.ModelKey == modelKey && r.RewardScore > 0 {
			expiresAt := r.ExpiresAt
			if expiresAt != nil && now.After(*expiresAt) {
				continue
			}
			totalReward += r.CurrentScore
		}
	}

	return totalPenalty, totalReward, nil
}

func (s *ExtraRatingService) CleanupExpired() error {
	return s.repo.DeleteExpiredRecords()
}

func NormalizeModelKey(channelName, format, modelType, modelName string) string {
	key := strings.ToLower(fmt.Sprintf("%s_%s_%s_%s", channelName, format, modelType, modelName))
	return key
}

func (s *ExtraRatingService) ApplyPenalty(modelKey string, score int, decayPerRequest int) error {
	if score > 0 {
		score = -score
	}
	if decayPerRequest <= 0 {
		decayPerRequest = 1
	}
	return s.repo.AddPenaltyRecord(modelKey, score, decayPerRequest, 0, nil)
}

func (s *ExtraRatingService) ApplyReward(modelKey string, score int, hours int) error {
	if score <= 0 || hours <= 0 {
		return fmt.Errorf("invalid reward parameters")
	}
	expiresAt := time.Now().Add(time.Duration(hours) * time.Hour)
	return s.repo.AddRewardRecord(modelKey, score, 0, 0, &expiresAt)
}

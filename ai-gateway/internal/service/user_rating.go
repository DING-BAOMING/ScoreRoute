package service

import (
	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type UserRatingService struct {
	repo *repository.UserRatingRepo
}

func NewUserRatingService() *UserRatingService {
	return &UserRatingService{
		repo: repository.NewUserRatingRepo(),
	}
}

func (s *UserRatingService) UpsertRating(req *model.UserRatingRequest) error {
	return s.repo.Upsert(req.ModelName, req.UserRating)
}

func (s *UserRatingService) GetRating(modelName string) (*model.UserRating, error) {
	return s.repo.GetByName(modelName)
}

func (s *UserRatingService) ListRatings() ([]*model.UserRating, error) {
	return s.repo.List()
}

func (s *UserRatingService) DeleteRating(id int64) error {
	return s.repo.Delete(id)
}

func (s *UserRatingService) GetAllUserRatings() (map[string]int, error) {
	return s.repo.GetAllAsMap()
}

func (s *UserRatingService) GetDeduplicatedModelNames() ([]string, error) {
	return s.repo.GetDeduplicatedModelNames()
}

func (s *UserRatingService) GetAllUserRatingsWithDefaults() ([]map[string]interface{}, error) {
	return s.repo.GetAllUserRatings()
}

func (s *UserRatingService) GetDeduplicatedRatings() ([]map[string]interface{}, error) {
	return s.repo.GetDeduplicatedUserRatings()
}

func (s *UserRatingService) GetNormalizedRating(modelName string) (int, error) {
	return s.repo.GetUserRatingForNormalizedModel(modelName)
}

func (s *UserRatingService) NormalizeModelName(name string) string {
	return s.repo.NormalizeModelName(name)
}

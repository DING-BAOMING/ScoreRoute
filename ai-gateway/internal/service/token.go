package service

import (
	"fmt"

	"github.com/google/uuid"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type TokenService struct {
	repo *repository.TokenRepo
}

func NewTokenService() *TokenService {
	return &TokenService{
		repo: repository.NewTokenRepo(),
	}
}

func (s *TokenService) Create(req *model.TokenRequest) (*model.Token, error) {
	if req.RateLimits == "" {
		req.RateLimits = "[]"
	}
	if req.Enabled == 0 {
		req.Enabled = 1
	}
	if req.Key == "" {
		req.Key = "sk-" + uuid.New().String()
	}
	token, err := s.repo.Create(req)
	if err != nil {
		return nil, err
	}
	token.Key = req.Key
	return token, nil
}

func (s *TokenService) GetByID(id int64) (*model.Token, error) {
	return s.repo.GetByID(id)
}

func (s *TokenService) GetByKey(key string) (*model.Token, error) {
	return s.repo.GetByKey(key)
}

func (s *TokenService) List(page, pageSize int) ([]*model.Token, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize)
}

func (s *TokenService) SetEnabled(id int64, enabled int) error {
	return s.repo.SetEnabled(id, enabled)
}

func (s *TokenService) Update(id int64, req *model.TokenRequest) (*model.Token, error) {
	if req.RateLimits == "" {
		req.RateLimits = "[]"
	}
	return s.repo.Update(id, req)
}

func (s *TokenService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *TokenService) RegenerateKey(id int64) (*model.Token, error) {
	oldToken, err := s.repo.GetByID(id)
	if err != nil || oldToken == nil {
		return nil, fmt.Errorf("token not found")
	}

	req := &model.TokenRequest{
		Name:            oldToken.Name,
		Format:          oldToken.Format,
		Type:            oldToken.Type,
		ModelName:       oldToken.ModelName,
		Enabled:         oldToken.Enabled,
		RateLimits:      oldToken.RateLimits,
		TotalTokenLimit: oldToken.TotalTokenLimit,
		ExpiresAt:       oldToken.ExpiresAt,
	}

	newKey := "sk-" + uuid.New().String()
	req.Key = newKey

	newToken, err := s.repo.Create(req)
	if err != nil {
		return nil, err
	}
	newToken.Key = newKey

	s.repo.Delete(id)

	return newToken, nil
}

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
	key := "sk-" + uuid.New().String()
	token, err := s.repo.Create(key, req.Name, req.Format, req.Type, req.ModelName)
	if err != nil {
		return nil, err
	}
	token.Key = key
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
	return s.repo.Update(id, req.Name, req.Format, req.Type, req.ModelName)
}

func (s *TokenService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *TokenService) RegenerateKey(id int64) (*model.Token, error) {
	token, err := s.repo.GetByID(id)
	if err != nil || token == nil {
		return nil, fmt.Errorf("token not found")
	}

	newKey := "sk-" + uuid.New().String()
	_, err = s.repo.Create(newKey, token.Name, token.Format, token.Type, token.ModelName)
	if err != nil {
		return nil, err
	}

	s.repo.Delete(id)

	token, err = s.repo.GetByKey(newKey)
	if err != nil || token == nil {
		return nil, fmt.Errorf("token not found")
	}
	token.Key = newKey
	return token, nil
}

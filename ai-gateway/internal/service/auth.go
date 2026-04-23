package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ai-gateway/internal/config"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(username, password string) (string, error) {
	if username != "admin" {
		return "", errors.New("用户名或密码错误")
	}

	if password != config.AppConfig.AdminPassword {
		return "", errors.New("用户名或密码错误")
	}

	return s.GenerateTokenForUser(username)
}

func (s *AuthService) GenerateTokenForUser(username string) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JwtSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

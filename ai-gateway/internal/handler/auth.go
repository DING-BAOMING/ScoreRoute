package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"ai-gateway/internal/service"
)

type AuthHandler struct {
	service    *service.AuthService
	configRepo *repository.SystemConfigRepo
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		service:    service.NewAuthService(),
		configRepo: repository.NewSystemConfigRepo(),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "登录成功",
		Data:    model.LoginResponse{Token: token},
	})
}

func (h *AuthHandler) Validate(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "未认证"})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "认证成功",
		Data:    gin.H{"username": username},
	})
}

func (h *AuthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ScoreRoute",
	})
}

func (h *AuthHandler) CheckSetupStatus(c *gin.Context) {
	config, err := h.configRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"password_setup_done": config.PasswordSetupDone,
			"password_less_mode":  config.PasswordLessMode,
		},
	})
}

func (h *AuthHandler) PasswordLessLogin(c *gin.Context) {
	config, err := h.configRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	if !config.PasswordLessMode {
		c.JSON(http.StatusUnauthorized, model.APIResponse{Code: 401, Message: "密码登录已启用，请使用密码登录"})
		return
	}

	token, err := h.service.GenerateTokenForUser("admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "登录成功",
		Data:    model.LoginResponse{Token: token},
	})
}

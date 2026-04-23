package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type SystemConfigHandler struct {
	repo *repository.SystemConfigRepo
}

func NewSystemConfigHandler() *SystemConfigHandler {
	return &SystemConfigHandler{
		repo: repository.NewSystemConfigRepo(),
	}
}

func (h *SystemConfigHandler) Get(c *gin.Context) {
	config, err := h.repo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: config})
}

func (h *SystemConfigHandler) Update(c *gin.Context) {
	var req struct {
		ExchangeRate     float64 `json:"exchange_rate" binding:"required"`
		Currency         string  `json:"currency" binding:"required"`
		PasswordLessMode bool    `json:"password_less_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.repo.Update(req.ExchangeRate, req.Currency, req.PasswordLessMode); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

func (h *SystemConfigHandler) UpdateDispatchMode(c *gin.Context) {
	var req struct {
		DispatchMode string `json:"dispatch_mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if req.DispatchMode != "polling" && req.DispatchMode != "smart" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "dispatch_mode must be 'polling' or 'smart'"})
		return
	}

	if err := h.repo.UpdateDispatchMode(req.DispatchMode); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

func (h *SystemConfigHandler) SetupPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "密码不能为空"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "密码长度至少6位"})
		return
	}

	if err := h.repo.SetupPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "密码设置成功"})
}

func (h *SystemConfigHandler) EnablePasswordLessMode(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.repo.EnablePasswordLessMode(req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "设置成功"})
}

func (h *SystemConfigHandler) CheckSetupStatus(c *gin.Context) {
	config, err := h.repo.Get()
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

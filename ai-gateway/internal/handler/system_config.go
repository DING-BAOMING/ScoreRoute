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
		ExchangeRate float64 `json:"exchange_rate" binding:"required"`
		Currency    string   `json:"currency" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.repo.Update(req.ExchangeRate, req.Currency); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

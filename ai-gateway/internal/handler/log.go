package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type LogHandler struct {
	service *service.LogService
}

func NewLogHandler() *LogHandler {
	return &LogHandler{
		service: service.NewLogService(),
	}
}

func (h *LogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var startTime, endTime *time.Time
	if start := c.Query("start_time"); start != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", start); err == nil {
			startTime = &t
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", end); err == nil {
			endTime = &t
		}
	}

	modelFilter := c.Query("model")
	statusFilter := c.Query("status")
	tokenIdFilter := c.Query("token_id")
	channelIdFilter := c.Query("channel_id")

	logs, total, err := h.service.List(page, pageSize, startTime, endTime, modelFilter, statusFilter, tokenIdFilter, channelIdFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查询成功",
		Data:    model.PageResult{Total: total, Items: logs},
	})
}

func (h *LogHandler) Stats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: stats})
}

func (h *LogHandler) Dashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: dashboard})
}

func (h *LogHandler) Cleanup(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}

	// Try to bind from query parameter first (for DELETE /cleanup?days=30)
	daysStr := c.Query("days")
	if daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			req.Days = days
		}
	}

	// If no days from query, try to bind from JSON body
	if req.Days == 0 {
		if err := c.ShouldBindJSON(&req); err != nil || req.Days <= 0 {
			c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的天数"})
			return
		}
	}

	if err := h.service.Cleanup(req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "清理成功"})
}

func (h *LogHandler) ModelStats(c *gin.Context) {
	stats, err := h.service.GetModelStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: stats})
}

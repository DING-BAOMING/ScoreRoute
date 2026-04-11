package handler

import (
	"net/http"
	"strconv"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

type SampleHandler struct {
	service *service.SampleService
}

func NewSampleHandler() *SampleHandler {
	return &SampleHandler{
		service: service.NewSampleService(),
	}
}

func (h *SampleHandler) List(c *gin.Context) {
	samples, err := h.service.ListSamples()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	for _, s := range samples {
		s.RemainingDays = service.GetRemainingDays(s.ExpiresAt)
		s.RemainingMinutes = service.GetRemainingMinutes(s.ExpiresAt)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: samples})
}

func (h *SampleHandler) Get(c *gin.Context) {
	modelKey := c.Param("modelKey")
	if modelKey == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "model_key is required"})
		return
	}

	sample, err := h.service.GetByModelKey(modelKey)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "Sample not found"})
		return
	}

	if sample == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "Sample not found"})
		return
	}

	sample.RemainingDays = service.GetRemainingDays(sample.ExpiresAt)
	sample.RemainingMinutes = service.GetRemainingMinutes(sample.ExpiresAt)
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: sample})
}

func (h *SampleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "Invalid ID"})
		return
	}

	if err := h.service.DeleteSample(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "删除成功"})
}

func (h *SampleHandler) Stats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: stats})
}

func (h *SampleHandler) Cleanup(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Days <= 0 {
		req.Days = 7
	}

	affected, err := h.service.CleanupOldSamples(req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "清理成功", Data: map[string]interface{}{"deleted": affected}})
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type ModelHandler struct {
	service            *service.ModelService
	extraRatingService *service.ExtraRatingService
}

func NewModelHandler() *ModelHandler {
	return &ModelHandler{
		service:            service.NewModelService(),
		extraRatingService: service.NewExtraRatingService(),
	}
}

func (h *ModelHandler) Create(c *gin.Context) {
	var req model.ModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	modelItem, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	modelKey := service.NormalizeModelKey(modelItem.ChannelName, modelItem.Format, modelItem.Type, modelItem.Name)
	h.extraRatingService.ApplyNewModelReward(modelKey)

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "创建成功", Data: modelItem})
}

func (h *ModelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "模型不存在"})
		return
	}

	var req model.ModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	fullReq := model.ModelRequest{
		ChannelID:       req.ChannelID,
		Name:            req.Name,
		Type:            req.Type,
		Enabled:         req.Enabled,
		RateLimits:      req.RateLimits,
		TotalTokenLimit: req.TotalTokenLimit,
		ExpiresAt:       req.ExpiresAt,
		CostPerToken:    req.CostPerToken,
		Currency:        req.Currency,
	}
	if fullReq.Name == "" {
		fullReq.Name = existing.Name
	}
	if fullReq.Type == "" {
		fullReq.Type = existing.Type
	}
	if fullReq.ChannelID == 0 {
		fullReq.ChannelID = existing.ChannelID
	}

	modelItem, err := h.service.Update(id, &fullReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功", Data: modelItem})
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "删除成功"})
}

func (h *ModelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	modelItem, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: modelItem})
}

func (h *ModelHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	models, total, err := h.service.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查询成功",
		Data:    model.PageResult{Total: total, Items: models},
	})
}

func (h *ModelHandler) ListByChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("channel_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的渠道ID"})
		return
	}

	models, err := h.service.ListByChannel(channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: models})
}

func (h *ModelHandler) SetEnabled(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	var req struct {
		Enabled int `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.service.SetEnabled(id, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "设置成功"})
}

func (h *ModelHandler) BatchCreate(c *gin.Context) {
	var req struct {
		ChannelID  int64    `json:"channel_id" binding:"required"`
		ModelNames []string `json:"model_names"`
		Models     []struct {
			Name      string `json:"name"`
			ModelName string `json:"model_name"`
			Type      string `json:"type"`
		} `json:"models"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if len(req.ModelNames) == 0 && len(req.Models) > 0 {
		for _, m := range req.Models {
			modelType := m.Type
			if modelType == "" {
				modelType = req.Type
			}
			if modelType == "" {
				modelType = "chat"
			}
			modelName := m.ModelName
			if modelName == "" {
				modelName = m.Name
			}
			req.ModelNames = append(req.ModelNames, modelName+"||"+modelType+"||"+m.Name)
		}
	}

	if len(req.ModelNames) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "model_names is required"})
		return
	}

	modelType := req.Type
	if modelType == "" {
		modelType = "chat"
	}

	createdKeys, err := h.service.BatchCreateWithDetails(req.ChannelID, req.ModelNames, modelType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	for _, key := range createdKeys {
		h.extraRatingService.ApplyNewModelReward(key)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "批量创建成功", Data: len(createdKeys)})
}

func (h *ModelHandler) Test(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	result, err := h.service.TestModel(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "测试完成", Data: result})
}

func (h *ModelHandler) SetRateLimit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "模型不存在"})
		return
	}

	var req struct {
		RateLimits string `json:"rate_limits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	fullReq := model.ModelRequest{
		ChannelID:  existing.ChannelID,
		Name:       existing.Name,
		Type:       existing.Type,
		Enabled:    existing.Enabled,
		RateLimits: req.RateLimits,
	}

	modelItem, err := h.service.Update(id, &fullReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "设置成功", Data: modelItem})
}

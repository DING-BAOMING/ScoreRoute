package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type TokenHandler struct {
	service *service.TokenService
}

func NewTokenHandler() *TokenHandler {
	return &TokenHandler{
		service: service.NewTokenService(),
	}
}

func (h *TokenHandler) Create(c *gin.Context) {
	var req model.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	token, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "创建成功", Data: token})
}

func (h *TokenHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	token, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: token})
}

func (h *TokenHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tokens, total, err := h.service.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查询成功",
		Data:    model.PageResult{Total: total, Items: tokens},
	})
}

func (h *TokenHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "Token不存在"})
		return
	}

	var req model.TokenUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	fullReq := model.TokenRequest{
		Key:             req.Key,
		Name:            req.Name,
		Format:          req.Format,
		Type:            req.Type,
		ModelName:       req.ModelName,
		Enabled:         req.Enabled,
		RateLimits:      req.RateLimits,
		TotalTokenLimit: req.TotalTokenLimit,
		ExpiresAt:       req.ExpiresAt,
	}
	if fullReq.Name == "" {
		fullReq.Name = existing.Name
	}
	if fullReq.Format == "" {
		fullReq.Format = existing.Format
	}
	if fullReq.Type == "" {
		fullReq.Type = existing.Type
	}

	token, err := h.service.Update(id, &fullReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功", Data: token})
}

func (h *TokenHandler) SetEnabled(c *gin.Context) {
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

func (h *TokenHandler) Delete(c *gin.Context) {
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

func (h *TokenHandler) RegenerateKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	token, err := h.service.RegenerateKey(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "密钥已重新生成", Data: token})
}

func (h *TokenHandler) SetRateLimit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	existing, err := h.service.GetByID(id)
	if err != nil || existing == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{Code: 404, Message: "Token不存在"})
		return
	}

	var req struct {
		RateLimits any    `json:"rate_limits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	rateLimitsStr := ""
	switch v := req.RateLimits.(type) {
	case string:
		rateLimitsStr = v
	case []interface{}, map[string]interface{}:
		data, _ := json.Marshal(v)
		rateLimitsStr = string(data)
	}

	fullReq := model.TokenRequest{
		Name:       existing.Name,
		Format:     existing.Format,
		Type:       existing.Type,
		ModelName:  existing.ModelName,
		Enabled:    existing.Enabled,
		RateLimits: rateLimitsStr,
	}

	token, err := h.service.Update(id, &fullReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "设置成功", Data: token})
}

func (h *TokenHandler) BatchCreate(c *gin.Context) {
	var req struct {
		Tokens []model.TokenRequest `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	results := make([]*model.Token, 0, len(req.Tokens))
	for _, tokenReq := range req.Tokens {
		token, err := h.service.Create(&tokenReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
			return
		}
		results = append(results, token)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "批量创建成功", Data: results})
}

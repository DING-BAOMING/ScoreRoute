package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type ExtraRatingHandler struct {
	service *service.ExtraRatingService
}

func NewExtraRatingHandler() *ExtraRatingHandler {
	return &ExtraRatingHandler{
		service: service.NewExtraRatingService(),
	}
}

func (h *ExtraRatingHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: config})
}

func (h *ExtraRatingHandler) SetConfig(c *gin.Context) {
	var req struct {
		PunishmentRounds int `json:"punishment_rounds"`
		PunishmentScore  int `json:"punishment_score"`
		RewardHours      int `json:"reward_hours"`
		RewardScore      int `json:"reward_score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	config := &service.ExtraRatingConfig{
		PunishmentRounds: req.PunishmentRounds,
		PunishmentScore:  req.PunishmentScore,
		RewardHours:      req.RewardHours,
		RewardScore:      req.RewardScore,
	}

	if err := h.service.SetConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

func (h *ExtraRatingHandler) GetRecords(c *gin.Context) {
	records, err := h.service.GetRecords()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: records})
}

func (h *ExtraRatingHandler) ClearRecords(c *gin.Context) {
	if err := h.service.ClearRecords(); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "清除成功"})
}

func (h *ExtraRatingHandler) DeleteRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	if err := h.service.DeleteRecord(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "删除成功"})
}

func (h *ExtraRatingHandler) GetAllModelExtraScores(c *gin.Context) {
	penaltyRecords, err := h.service.GetPenaltyRecords()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	rewardRecords, err := h.service.GetRewardRecords()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	penaltyMap := make(map[string]int)
	for _, p := range penaltyRecords {
		if p.CurrentScore < 0 {
			penaltyMap[p.ModelKey] = penaltyMap[p.ModelKey] + p.CurrentScore
		}
	}

	rewardMap := make(map[string]int)
	now := time.Now()
	for _, r := range rewardRecords {
		if r.RewardScore > 0 && r.ExpiresAt != nil && r.CreatedAt.Unix() > 0 {
			totalDuration := r.ExpiresAt.Sub(r.CreatedAt).Minutes()
			elapsed := now.Sub(r.CreatedAt).Minutes()
			if elapsed < totalDuration {
				remainingRatio := 1 - elapsed/totalDuration
				rewardValue := float64(r.RewardScore) * remainingRatio
				rewardMap[r.ModelKey] = rewardMap[r.ModelKey] + int(rewardValue)
			}
		}
	}

	result := make(map[string]map[string]int)
	for key, penalty := range penaltyMap {
		if result[key] == nil {
			result[key] = map[string]int{"penalty": 0, "reward": 0}
		}
		result[key]["penalty"] = penalty
	}
	for key, reward := range rewardMap {
		if result[key] == nil {
			result[key] = map[string]int{"penalty": 0, "reward": 0}
		}
		result[key]["reward"] = reward
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: result})
}

func (h *ExtraRatingHandler) UpdatePenalty(c *gin.Context) {
	var req struct {
		ModelKey        string `json:"model_key" binding:"required"`
		Score           int    `json:"score"`
		DecayPerRequest int    `json:"decay_per_request"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.service.ApplyPenalty(req.ModelKey, req.Score, req.DecayPerRequest); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "惩罚已应用"})
}

func (h *ExtraRatingHandler) UpdateReward(c *gin.Context) {
	var req struct {
		ModelKey string `json:"model_key" binding:"required"`
		Score    int    `json:"score"`
		Hours    int    `json:"hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if err := h.service.ApplyReward(req.ModelKey, req.Score, req.Hours); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "奖励已应用"})
}

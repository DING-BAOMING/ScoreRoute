package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type ExtraRatingHandler struct {
	service      *service.ExtraRatingService
	modelService *service.ModelService
}

func NewExtraRatingHandler() *ExtraRatingHandler {
	return &ExtraRatingHandler{
		service:      service.NewExtraRatingService(),
		modelService: service.NewModelService(),
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

	response := &model.ExtraRatingResponse{
		PenaltyRecords: make([]model.ExtraRatingRecord, 0),
		RewardRecords:  make([]model.ExtraRatingRecord, 0),
	}

	for _, p := range penaltyRecords {
		response.PenaltyRecords = append(response.PenaltyRecords, *p)
	}
	for _, r := range rewardRecords {
		response.RewardRecords = append(response.RewardRecords, *r)
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: response})
}

type PenaltyRequest struct {
	ModelKey        string `json:"model_key"`
	ModelName       string `json:"model_name"`
	Score           int    `json:"score"`
	DecayPerRequest int    `json:"decay_per_request"`
	Penalty         int    `json:"penalty"`
	PenaltyScore    int    `json:"penalty_score"`
}

func (r *PenaltyRequest) ParseAndValidate(h *ExtraRatingHandler) (string, int, int, error) {
	modelKey := r.ModelKey

	// If model_key is empty but model_name is provided, try to look up model_key
	if modelKey == "" && r.ModelName != "" {
		models, err := h.modelService.GetByNamePrefix(r.ModelName)
		if err == nil && len(models) > 0 {
			// Use the first matching model
			model := models[0]
			modelKey = normalizeModelKey(model.ChannelName, model.Format, model.Type, model.Name)
		}
	}

	if modelKey == "" {
		return "", 0, 0, &json.UnmarshalTypeError{}
	}

	score := r.Score
	if score == 0 && r.Penalty > 0 {
		score = r.Penalty
	}
	if score == 0 && r.PenaltyScore > 0 {
		score = r.PenaltyScore
	}

	if score <= 0 {
		return "", 0, 0, &json.UnmarshalTypeError{}
	}

	decay := r.DecayPerRequest
	if decay <= 0 {
		decay = 1
	}

	return modelKey, score, decay, nil
}

func (h *ExtraRatingHandler) UpdatePenalty(c *gin.Context) {
	var req PenaltyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误：需要 model_key 和 score/penalty/penalty_score"})
		return
	}

	modelKey, score, decay, err := req.ParseAndValidate(h)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "惩罚分数必须在1-100之间，请使用 score、penalty 或 penalty_score 字段"})
		return
	}

	if err := h.service.ApplyPenalty(modelKey, score, decay); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "惩罚已应用"})
}

type RewardRequest struct {
	ModelKey    string `json:"model_key"`
	ModelName   string `json:"model_name"`
	Score       int    `json:"score"`
	Hours       int    `json:"hours"`
	Reward      int    `json:"reward"`
	RewardScore int    `json:"reward_score"`
	Reason      string `json:"reason"`
}

func (r *RewardRequest) ParseAndValidate(h *ExtraRatingHandler) (string, int, int, error) {
	modelKey := r.ModelKey

	// If model_key is empty but model_name is provided, try to look up model_key
	if modelKey == "" && r.ModelName != "" {
		models, err := h.modelService.GetByNamePrefix(r.ModelName)
		if err == nil && len(models) > 0 {
			// Use the first matching model
			model := models[0]
			modelKey = normalizeModelKey(model.ChannelName, model.Format, model.Type, model.Name)
		}
	}

	if modelKey == "" {
		return "", 0, 0, &json.UnmarshalTypeError{}
	}

	score := r.Score
	if score == 0 && r.Reward > 0 {
		score = r.Reward
	}
	if score == 0 && r.RewardScore > 0 {
		score = r.RewardScore
	}

	hours := r.Hours
	if hours <= 0 {
		hours = 24
	}

	if score <= 0 {
		return "", 0, 0, &json.UnmarshalTypeError{}
	}

	return modelKey, score, hours, nil
}

func normalizeModelKey(channelName, format, modelType, modelName string) string {
	name := strings.ToLower(channelName) + "_" + strings.ToLower(format) + "_" + strings.ToLower(modelType) + "_" + strings.ToLower(modelName)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func (h *ExtraRatingHandler) UpdateReward(c *gin.Context) {
	var req RewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误：需要 model_key 和 score/reward/reward_score"})
		return
	}

	modelKey, score, hours, err := req.ParseAndValidate(h)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "奖励分数必须在1-100之间，请使用 score、reward 或 reward_score 字段"})
		return
	}

	if err := h.service.ApplyReward(modelKey, score, hours); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "奖励已应用"})
}

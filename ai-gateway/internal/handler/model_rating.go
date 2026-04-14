package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"ai-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

var _ = fmt.Sprintf

type ModelRatingHandler struct {
	configRepo *repository.ModelRatingConfigRepo
	modelRepo  *repository.ModelRepo
	configSvc  *repository.SystemConfigRepo
	modelRatingSvc *service.ModelRatingService
}

func NewModelRatingHandler() *ModelRatingHandler {
	return &ModelRatingHandler{
		configRepo: repository.NewModelRatingConfigRepo(),
		modelRepo:  repository.NewModelRepo(),
		configSvc:  repository.NewSystemConfigRepo(),
		modelRatingSvc: service.NewModelRatingService(),
	}
}

func (h *ModelRatingHandler) GetWeights(c *gin.Context) {
	weights, err := h.configRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: weights})
}

func (h *ModelRatingHandler) UpdateWeights(c *gin.Context) {
	var req struct {
		SuccessWeight      float64 `json:"success_weight"`
		LatencyWeight     float64 `json:"latency_weight"`
		ReliabilityWeight float64 `json:"reliability_weight"`
		UserRatingWeight  float64 `json:"user_rating_weight"`
		SampleRatingWeight float64 `json:"sample_rating_weight"`
		CostRatingWeight  float64 `json:"cost_rating_weight"`
		TimeRatingWeight  float64 `json:"time_rating_weight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	totalWeight := req.SuccessWeight + req.LatencyWeight + req.ReliabilityWeight + req.UserRatingWeight + req.SampleRatingWeight + req.CostRatingWeight + req.TimeRatingWeight
	if totalWeight > 1.0 || totalWeight < 0.1 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "权重总和必须在0.1到1之间"})
		return
	}

	weights := &repository.ModelRatingWeights{
		SuccessWeight:      req.SuccessWeight,
		LatencyWeight:     req.LatencyWeight,
		ReliabilityWeight:  req.ReliabilityWeight,
		UserRatingWeight:   req.UserRatingWeight,
		SampleRatingWeight: req.SampleRatingWeight,
		CostRatingWeight:   req.CostRatingWeight,
		TimeRatingWeight:   req.TimeRatingWeight,
	}

	if err := h.configRepo.Update(weights); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

func (h *ModelRatingHandler) GetAllScores(c *gin.Context) {
	scores, err := h.modelRatingSvc.CalculateAllScores()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: scores})
}

type ModelCostTimeRating struct {
	ModelKey     string  `json:"model_key"`
	CostRating   int     `json:"cost_rating"`
	TimeRating   int     `json:"time_rating"`
	CostPerToken float64 `json:"cost_per_token"`
	Currency     string  `json:"currency"`
	ExpiresAt    string  `json:"expires_at,omitempty"`
	DaysLeft     int     `json:"days_left"`
}

func (h *ModelRatingHandler) GetCostTimeRatings(c *gin.Context) {
	models, err := h.modelRepo.ListEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	config, _ := h.configSvc.Get()
	exchangeRate := 7.2
	if config != nil {
		exchangeRate = config.ExchangeRate
	}

	ratings := make([]ModelCostTimeRating, 0, len(models))
	for _, m := range models {
		rateLimits := m.RateLimits
		channelRateLimits := m.ChannelRateLimits
		if rateLimits == "" || rateLimits == "[]" {
			rateLimits = channelRateLimits
		}
		costRating := h.calculateCostRating(m.CostPerToken, m.Currency, rateLimits, channelRateLimits, exchangeRate, models)
		timeRating := h.calculateTimeRating(m.ExpiresAt)

		expiresAtStr := ""
		daysLeft := 0
		if m.ExpiresAt != nil {
			expiresAtStr = m.ExpiresAt.Format("2006-01-02 15:04:05")
			daysLeft = int(time.Until(*m.ExpiresAt).Hours() / 24)
			if daysLeft < 0 {
				daysLeft = 0
			}
		}

		modelKey := NormalizeModelKey(m.ChannelName, m.Format, m.Type, m.Name)
		ratings = append(ratings, ModelCostTimeRating{
			ModelKey:     modelKey,
			CostRating:   costRating,
			TimeRating:   timeRating,
			CostPerToken: m.CostPerToken,
			Currency:     m.Currency,
			ExpiresAt:    expiresAtStr,
			DaysLeft:     daysLeft,
		})
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: ratings})
}

func (h *ModelRatingHandler) calculateCostRating(costPerToken float64, currency string, rateLimits string, channelRateLimits string, exchangeRate float64, allModels []*model.Model) int {
	if isPeriodicBilling(rateLimits) || isPeriodicBilling(channelRateLimits) {
		return 100
	}

	if costPerToken <= 0 {
		return 90
	}

	convertedCost := costPerToken
	if currency == "USD" {
		convertedCost = costPerToken * exchangeRate
	}

	var paidCosts []float64
	for _, m := range allModels {
		mRateLimits := m.RateLimits
		if m.RateLimits == "" || m.RateLimits == "[]" {
			mRateLimits = m.ChannelRateLimits
		}
		if m.CostPerToken > 0 && !isPeriodicBilling(mRateLimits) {
			c := m.CostPerToken
			if m.Currency == "USD" {
				c = m.CostPerToken * exchangeRate
			}
			paidCosts = append(paidCosts, c)
		}
	}

	if len(paidCosts) == 0 {
		return 90
	}

	minCost := paidCosts[0]
	maxCost := paidCosts[0]
	for _, c := range paidCosts {
		if c < minCost {
			minCost = c
		}
		if c > maxCost {
			maxCost = c
		}
	}

	if len(paidCosts) == 1 || minCost == maxCost {
		return 50
	}

	ratio := (convertedCost - minCost) / (maxCost - minCost)
	score := 70 - int(ratio*69)
	if score < 1 {
		score = 1
	}
	if score > 70 {
		score = 70
	}

	return score
}

func isPeriodicBilling(rateLimits string) bool {
	if rateLimits == "" || rateLimits == "[]" {
		return false
	}
	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(rateLimits), &rules); err != nil {
		return false
	}
	for _, rule := range rules {
		if rule.Type == "billing" {
			return true
		}
	}
	return false
}

func (h *ModelRatingHandler) calculateTimeRating(expiresAt *time.Time) int {
	if expiresAt == nil {
		return 70
	}

	daysLeft := time.Until(*expiresAt).Hours() / 24
	if daysLeft < 0 {
		daysLeft = 0
	}

	if daysLeft < 7 {
		return 100
	}
	if daysLeft < 30 {
		ratio := (daysLeft - 7) / 23.0
		return 100 - int(ratio*10)
	}
	if daysLeft < 60 {
		ratio := (daysLeft - 30) / 30.0
		return 90 - int(ratio*10)
	}
	if daysLeft < 120 {
		ratio := (daysLeft - 60) / 60.0
		return 80 - int(ratio*10)
	}
	if daysLeft < 180 {
		ratio := (daysLeft - 120) / 60.0
		return 70 - int(ratio*10)
	}
	if daysLeft < 365 {
		ratio := (daysLeft - 180) / 185.0
		return 60 - int(ratio*59)
	}
	return 0
}

func NormalizeModelKey(channelName, format, modelType, modelName string) string {
	key := fmt.Sprintf("%s_%s_%s_%s", channelName, format, modelType, modelName)
	return strings.ToLower(key)
}

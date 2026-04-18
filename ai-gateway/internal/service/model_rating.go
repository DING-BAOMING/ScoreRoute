package service

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/repository"

	"ai-gateway/internal/model"
)

const defaultExchangeRate = 7.2

type ModelRatingService struct {
	modelRepo        *repository.ModelRepo
	logRepo          *repository.LogRepo
	userRatingRepo   *repository.UserRatingRepo
	sampleRatingRepo *repository.SampleRatingRepo
	extraRatingSvc   *ExtraRatingService
	systemConfigRepo *repository.SystemConfigRepo
}

func NewModelRatingService() *ModelRatingService {
	return &ModelRatingService{
		modelRepo:        repository.NewModelRepo(),
		logRepo:          repository.NewLogRepo(),
		userRatingRepo:   repository.NewUserRatingRepo(),
		sampleRatingRepo: repository.NewSampleRatingRepo(),
		extraRatingSvc:   NewExtraRatingService(),
		systemConfigRepo: repository.NewSystemConfigRepo(),
	}
}

type ModelScore struct {
	ChannelName  string  `json:"channel_name"`
	Format       string  `json:"format"`
	ModelType    string  `json:"model_type"`
	ModelName    string  `json:"model_name"`
	ModelKey     string  `json:"model_key"`
	Score        float64 `json:"score"`
	SuccessRate  float64 `json:"success_rate"`
	Latency      float64 `json:"latency"`
	Reliability  float64 `json:"reliability"`
	UserRating   int     `json:"user_rating"`
	SampleRating int     `json:"sample_rating"`
	Penalty      int     `json:"penalty"`
	Reward       int     `json:"reward"`
	Rank         int     `json:"rank"`
	CallCount    int     `json:"call_count"`
}

type RatingWeights struct {
	SuccessWeight      float64 `json:"success_weight"`
	LatencyWeight      float64 `json:"latency_weight"`
	ReliabilityWeight  float64 `json:"reliability_weight"`
	UserRatingWeight   float64 `json:"user_rating_weight"`
	SampleRatingWeight float64 `json:"sample_rating_weight"`
	CostRatingWeight   float64 `json:"cost_rating_weight"`
	TimeRatingWeight   float64 `json:"time_rating_weight"`
}

func (s *ModelRatingService) GetWeights() (*RatingWeights, error) {
	configRepo := repository.NewModelRatingConfigRepo()
	weights, err := configRepo.GetAll()
	if err != nil {
		return nil, err
	}
	return &RatingWeights{
		SuccessWeight:      weights.SuccessWeight,
		LatencyWeight:      weights.LatencyWeight,
		ReliabilityWeight:  weights.ReliabilityWeight,
		UserRatingWeight:   weights.UserRatingWeight,
		SampleRatingWeight: weights.SampleRatingWeight,
		CostRatingWeight:   weights.CostRatingWeight,
		TimeRatingWeight:   weights.TimeRatingWeight,
	}, nil
}

func (s *ModelRatingService) CalculateAllScores() ([]*ModelScore, error) {
	models, err := s.modelRepo.ListEnabled()
	if err != nil {
		return nil, err
	}

	statsMap := s.logRepo.GetModelStatsMap()

	userRatings := s.getUserRatingsMap()
	sampleRatings, err := s.sampleRatingRepo.GetAllAsMap()
	if err != nil {
		log.Printf("[CalculateAllScores] failed to get sample ratings: %v", err)
		sampleRatings = make(map[string]*model.SampleRating)
	}
	extraScores := s.getExtraScoresMap()
	costTimeRatings := s.getCostTimeRatingsMap(models)

	weights, err := s.GetWeights()
	if err != nil {
		weights = &RatingWeights{
			SuccessWeight:      0.15,
			LatencyWeight:      0.1,
			ReliabilityWeight:  0.1,
			UserRatingWeight:   0.15,
			SampleRatingWeight: 0.25,
			CostRatingWeight:   0.15,
			TimeRatingWeight:   0.1,
		}
	}

	var scores []*ModelScore
	for _, m := range models {
		stats := statsMap[modelStatsKey(m.ChannelName, m.Name)]

		var totalCalls, successCalls int64
		var avgLatency float64
		if stats != nil {
			totalCalls = stats.TotalCalls
			successCalls = stats.SuccessCalls
			avgLatency = stats.AvgLatency
		}

		modelKey := normalizeModelKey(m.ChannelName, m.Format, m.Type, m.Name)

		successRate := 85.0
		if totalCalls > 0 {
			successRate = float64(successCalls) / float64(totalCalls) * 100
		}

		latencyScore := 85.0
		if avgLatency > 0 {
			latencyScore = maxFloat(0, 1-(avgLatency/30000)) * 100
		}

		reliabilityScore := 85.0
		if totalCalls >= 30 {
			reliabilityScore = 100
		} else if totalCalls >= 10 {
			reliabilityScore = 80 + 20*float64(totalCalls-10)/20
		} else if totalCalls >= 5 {
			reliabilityScore = 50 + 30*float64(totalCalls-5)/5
		} else if totalCalls > 0 {
			reliabilityScore = 50
		}

		userRating := 85
		normalizedName := normalizeForUserRating(m.Name)
		if ur, ok := userRatings[normalizedName]; ok {
			userRating = ur
		} else if ur, ok := userRatings[m.Name]; ok {
			userRating = ur
		} else if ur, ok := userRatings[strings.ToLower(m.Name)]; ok {
			userRating = ur
		}

		sampleRating := 85
		if sr, ok := sampleRatings[modelKey]; ok {
			sampleRating = sr.Score
		}

		penalty := 0
		reward := 0
		if extra, ok := extraScores[modelKey]; ok {
			penalty = extra.penalty
			reward = extra.reward
		}

		costRating := 50
		timeRating := 50
		if ct, ok := costTimeRatings[modelKey]; ok {
			costRating = ct.cost
			timeRating = ct.time
		}

		score := (successRate * weights.SuccessWeight) +
			(latencyScore * weights.LatencyWeight) +
			(reliabilityScore * weights.ReliabilityWeight) +
			(float64(userRating) * weights.UserRatingWeight) +
			(float64(sampleRating) * weights.SampleRatingWeight) +
			(float64(costRating) * weights.CostRatingWeight) +
			(float64(timeRating) * weights.TimeRatingWeight) +
			float64(penalty+reward)

		scores = append(scores, &ModelScore{
			ChannelName:  m.ChannelName,
			Format:       m.Format,
			ModelType:    m.Type,
			ModelName:    m.Name,
			ModelKey:     modelKey,
			Score:        score,
			SuccessRate:  successRate,
			Latency:      latencyScore,
			Reliability:  reliabilityScore,
			UserRating:   userRating,
			SampleRating: sampleRating,
			Penalty:      penalty,
			Reward:       reward,
			CallCount:    int(m.CallCount),
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score > scores[j].Score
		}
		return scores[i].CallCount < scores[j].CallCount
	})

	for i, sc := range scores {
		sc.Rank = i + 1
	}

	return scores, nil
}

func (s *ModelRatingService) getUserRatingsMap() map[string]int {
	ratings := make(map[string]int)
	deduped, err := s.userRatingRepo.GetDeduplicatedUserRatings()
	if err != nil {
		log.Printf("[getUserRatingsMap] failed to get user ratings: %v", err)
		return ratings
	}
	for _, r := range deduped {
		modelName, _ := r["model_name"].(string)
		rating, _ := r["user_rating"].(int)
		ratings[strings.ToLower(modelName)] = rating
	}
	return ratings
}

func (s *ModelRatingService) getExtraScoresMap() map[string]struct{ penalty, reward int } {
	result := make(map[string]struct{ penalty, reward int })

	penaltyRecords, err := s.extraRatingSvc.GetPenaltyRecords()
	if err != nil {
		log.Printf("[getExtraScoresMap] failed to get penalty records: %v", err)
	} else {
		for _, p := range penaltyRecords {
			if p.CurrentScore < 0 {
				key := p.ModelKey
				if v, ok := result[key]; ok {
					v.penalty += p.CurrentScore
					result[key] = v
				} else {
					result[key] = struct{ penalty, reward int }{penalty: p.CurrentScore, reward: 0}
				}
			}
		}
	}

	rewardRecords, err := s.extraRatingSvc.GetRewardRecords()
	if err != nil {
		log.Printf("[getExtraScoresMap] failed to get reward records: %v", err)
	} else {
		for _, r := range rewardRecords {
			if r.RewardScore > 0 {
				key := r.ModelKey
				if v, ok := result[key]; ok {
					v.reward += r.RewardScore
					result[key] = v
				} else {
					result[key] = struct{ penalty, reward int }{penalty: 0, reward: r.RewardScore}
				}
			}
		}
	}

	return result
}

func (s *ModelRatingService) getCostTimeRatingsMap(models []*model.Model) map[string]struct{ cost, time int } {
	result := make(map[string]struct{ cost, time int })
	if len(models) == 0 {
		return result
	}

	config, _ := s.systemConfigRepo.Get()
	exchangeRate := defaultExchangeRate
	if config != nil && config.ExchangeRate > 0 {
		exchangeRate = config.ExchangeRate
	}

	var paidCosts []float64
	for _, m := range models {
		mRateLimits := m.RateLimits
		if mRateLimits == "" || mRateLimits == "[]" {
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

	for _, m := range models {
		modelKey := normalizeModelKey(m.ChannelName, m.Format, m.Type, m.Name)

		mRateLimits := m.RateLimits
		if mRateLimits == "" || mRateLimits == "[]" {
			mRateLimits = m.ChannelRateLimits
		}

		costRating := 50
		if isPeriodicBilling(mRateLimits) {
			costRating = 100
		} else if m.CostPerToken <= 0 {
			costRating = 90
		} else if len(paidCosts) > 1 {
			convertedCost := m.CostPerToken
			if m.Currency == "USD" {
				convertedCost = m.CostPerToken * exchangeRate
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
			if maxCost > minCost {
				ratio := (convertedCost - minCost) / (maxCost - minCost)
				costRating = 70 - int(ratio*69)
				if costRating < 1 {
					costRating = 1
				}
				if costRating > 70 {
					costRating = 70
				}
			}
		}

		timeRating := 70
		if m.ExpiresAt != nil {
			daysLeft := time.Until(*m.ExpiresAt).Hours() / 24
			if daysLeft < 0 {
				daysLeft = 0
			}
			if daysLeft < 7 {
				timeRating = 100
			} else if daysLeft < 30 {
				timeRating = 100 - int((daysLeft-7)/23.0*10)
			} else if daysLeft < 60 {
				timeRating = 90 - int((daysLeft-30)/30.0*10)
			} else if daysLeft < 120 {
				timeRating = 80 - int((daysLeft-60)/60.0*10)
			} else if daysLeft < 180 {
				timeRating = 70 - int((daysLeft-120)/60.0*10)
			} else if daysLeft < 365 {
				timeRating = 60 - int((daysLeft-180)/185.0*59)
			} else {
				timeRating = 0
			}
		}

		result[modelKey] = struct{ cost, time int }{cost: costRating, time: timeRating}
	}

	return result
}

func isPeriodicBilling(rateLimits string) bool {
	if rateLimits == "" || rateLimits == "[]" {
		return false
	}
	var rules []struct {
		Type string `json:"type"`
	}
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

func normalizeModelKey(channelName, format, modelType, modelName string) string {
	return strings.ToLower(channelName + "_" + format + "_" + modelType + "_" + modelName)
}

func modelStatsKey(channelName, modelName string) string {
	return strings.ToLower(channelName + "::" + modelName)
}

func normalizeForUserRating(modelName string) string {
	modelName = strings.ToLower(strings.TrimSpace(modelName))

	if strings.HasPrefix(modelName, "minimaxai/") {
		modelName = strings.TrimPrefix(modelName, "minimaxai/")
		modelName = strings.TrimPrefix(modelName, "minimax-")
		modelName = strings.TrimPrefix(modelName, "minimax")
		if len(modelName) > 1 && modelName[0] == 'm' && modelName[1] >= '0' && modelName[1] <= '9' {
			modelName = modelName[1:]
		}
		return "minimax-" + modelName
	}

	if strings.HasPrefix(modelName, "minimax-") || strings.HasPrefix(modelName, "minimax") {
		modelName = strings.TrimPrefix(modelName, "minimax-")
		modelName = strings.TrimPrefix(modelName, "minimax")
		if len(modelName) > 1 && modelName[0] == 'm' && modelName[1] >= '0' && modelName[1] <= '9' {
			modelName = modelName[1:]
		}
		return "minimax-" + modelName
	}

	vendorPrefixes := []string{"google/", "qwen/", "z-ai/", "anthropic/", "openai/", "meta/", "mistral/", "cohere/", "azure/", "aws/", "alibaba/", "baidu/", "tencent/"}
	for _, prefix := range vendorPrefixes {
		if strings.HasPrefix(modelName, prefix) {
			modelName = strings.TrimPrefix(modelName, prefix)
			break
		}
	}

	return modelName
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

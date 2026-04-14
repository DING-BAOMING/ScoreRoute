package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"

	"github.com/pkoukk/tiktoken-go"
)

type Dispatcher struct {
	channelService     *ChannelService
	modelService       *ModelService
	logRepo            *repository.LogRepo
	sampleRepo         *repository.SampleRepo
	channelRepo        *repository.ChannelRepo
	rateLimitRepo      *repository.RateLimitRepo
	modelRateLimitRepo *repository.ModelRateLimitRepo
	modelRepo          *repository.ModelRepo
	systemConfigRepo   *repository.SystemConfigRepo
	extraRatingService *ExtraRatingService
	client             *http.Client
}

func NewDispatcher() *Dispatcher {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Dispatcher{
		channelService:      NewChannelService(),
		modelService:        NewModelService(),
		logRepo:             repository.NewLogRepo(),
		sampleRepo:          repository.NewSampleRepo(),
		channelRepo:         repository.NewChannelRepo(),
		rateLimitRepo:       repository.NewRateLimitRepo(),
		modelRateLimitRepo:  repository.NewModelRateLimitRepo(),
		modelRepo:           repository.NewModelRepo(),
		systemConfigRepo:    repository.NewSystemConfigRepo(),
		extraRatingService:  NewExtraRatingService(),
		client: &http.Client{
			Timeout:   300 * time.Second,
			Transport: transport,
		},
	}
}

type RelayResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (d *Dispatcher) ListEnabledModels() ([]*model.Model, error) {
	return d.modelService.ListEnabled()
}

func (d *Dispatcher) GetNextModelSmart(format, modelType string) (*model.Model, error) {
	config, err := d.systemConfigRepo.Get()
	if err != nil {
		return nil, err
	}

	dispatchMode := "polling"
	if config != nil && config.DispatchMode != "" {
		dispatchMode = config.DispatchMode
	}

	if dispatchMode != "smart" {
		return d.modelService.GetNextModelGlobal(format, modelType)
	}

	loadRatings()

	models, err := d.modelService.ListEnabled()
	if err != nil || len(models) == 0 {
		return d.modelService.GetNextModelGlobal(format, modelType)
	}

	weights, err := d.getModelRatingWeights()
	if err != nil {
		weights = &modelRatingWeights{
			SuccessWeight:      0.15,
			LatencyWeight:      0.10,
			ReliabilityWeight:  0.10,
			UserRatingWeight:   0.15,
			SampleRatingWeight: 0.25,
			CostRatingWeight:   0.15,
			TimeRatingWeight:   0.10,
		}
	}

	type scoredModel struct {
		model      *model.Model
		compositeScore float64
	}

	scoredModels := make([]scoredModel, 0, len(models))
	for _, m := range models {
		if format != "" && m.Format != format {
			continue
		}
		if modelType != "" && m.Type != modelType {
			continue
		}

		score := d.calculateCompositeScore(m, weights)
		scoredModels = append(scoredModels, scoredModel{model: m, compositeScore: score})
	}

	if len(scoredModels) == 0 {
		return d.modelService.GetNextModelGlobal(format, modelType)
	}

	sort.Slice(scoredModels, func(i, j int) bool {
		return scoredModels[i].compositeScore > scoredModels[j].compositeScore
	})

	selected := scoredModels[0].model
	d.modelRepo.IncrementCallCount(selected.ID)
	d.channelRepo.IncrementCallCount(selected.ChannelID)

	return selected, nil
}

type modelRatingWeights struct {
	SuccessWeight      float64
	LatencyWeight      float64
	ReliabilityWeight  float64
	UserRatingWeight   float64
	SampleRatingWeight float64
	CostRatingWeight   float64
	TimeRatingWeight   float64
}

func (d *Dispatcher) getModelRatingWeights() (*modelRatingWeights, error) {
	repo := repository.NewModelRatingConfigRepo()
	repoWeights, err := repo.GetAll()
	if err != nil {
		return nil, err
	}

	w := &modelRatingWeights{
		SuccessWeight:      repoWeights.SuccessWeight,
		LatencyWeight:     repoWeights.LatencyWeight,
		ReliabilityWeight: repoWeights.ReliabilityWeight,
		UserRatingWeight:   repoWeights.UserRatingWeight,
		SampleRatingWeight: repoWeights.SampleRatingWeight,
		CostRatingWeight:   repoWeights.CostRatingWeight,
		TimeRatingWeight:   repoWeights.TimeRatingWeight,
	}

	return w, nil
}

func (d *Dispatcher) calculateCompositeScore(m *model.Model, weights *modelRatingWeights) float64 {
	modelStats, err := d.logRepo.GetModelStatsByChannelAndModel(m.ChannelName, m.Name)
	if err != nil {
		modelStats = nil
	}

	var totalCalls, successCalls int64
	var avgLatency float64
	if modelStats != nil {
		totalCalls = modelStats.TotalCalls
		successCalls = modelStats.SuccessCalls
		avgLatency = modelStats.AvgLatency
	}

	successScore := 0.0
	if totalCalls > 0 {
		successScore = float64(successCalls) / float64(totalCalls)
	}

	latencyScore := 0.0
	if avgLatency > 0 {
		latencyScore = 1.0 - (avgLatency / 30000.0)
		if latencyScore < 0 {
			latencyScore = 0
		}
	}

	reliabilityScore := 0.0
	if totalCalls >= 30 {
		reliabilityScore = 1.0
	} else if totalCalls >= 10 {
		reliabilityScore = 0.8 + 0.2*float64(totalCalls-10)/20.0
	} else if totalCalls >= 5 {
		reliabilityScore = 0.5 + 0.3*float64(totalCalls-5)/5.0
	} else if totalCalls > 0 {
		reliabilityScore = 0.5
	}

	userRating := 50
	normalizedKey := normalizeUserRatingKey(m.Name)
	if ur, ok := userRatings[normalizedKey]; ok {
		userRating = ur
	} else if ur, ok := userRatings[m.Name]; ok {
		userRating = ur
	} else if ur, ok := userRatings[strings.ToLower(m.Name)]; ok {
		userRating = ur
	} else {
		fallbackKey := normalizedKey
		fallbackKey = strings.Replace(fallbackKey, "glm", "glm-", 1)
		if ur, ok := userRatings[fallbackKey]; ok {
			userRating = ur
		} else {
			fallbackKey2 := strings.Replace(normalizedKey, "-", "", -1)
			if ur, ok := userRatings[fallbackKey2]; ok {
				userRating = ur
			}
		}
	}
	userScore := float64(userRating) / 100.0

	sampleRating := 50
	sampleKey := NormalizeModelKey(m.ChannelName, m.Format, m.Type, m.Name)
	if sr, ok := sampleRatings[sampleKey]; ok {
		sampleRating = sr
	}
	sampleScore := float64(sampleRating) / 100.0

	penalty, reward, _ := d.extraRatingService.GetModelExtraScore(sampleKey)
	extraScore := float64(reward+penalty) / 100.0

	costRating := 90
	if m.CostPerToken > 0 {
		costRating = 70
	}
	costScore := float64(costRating) / 100.0

	timeRating := 70
	if m.ExpiresAt != nil {
		daysLeft := time.Until(*m.ExpiresAt).Hours() / 24
		if daysLeft < 7 {
			timeRating = 100
		} else if daysLeft < 30 {
			timeRating = 100 - int((daysLeft-7)/23.0*10)
		}
	}
	timeScore := float64(timeRating) / 100.0

	compositeScore := successScore*weights.SuccessWeight +
		latencyScore*weights.LatencyWeight +
		reliabilityScore*weights.ReliabilityWeight +
		userScore*weights.UserRatingWeight +
		sampleScore*weights.SampleRatingWeight +
		costScore*weights.CostRatingWeight +
		timeScore*weights.TimeRatingWeight +
		extraScore

	log.Printf("[calculateCompositeScore] model=%s/%s: score=%.4f (success=%.4f, latency=%.4f, reliability=%.4f, user=%d, sample=%d, extra=%.4f)",
		m.ChannelName, m.Name, compositeScore, successScore, latencyScore, reliabilityScore, userRating, sampleRating, extraScore)

	return compositeScore
}

var userRatings = make(map[string]int)
var sampleRatings = make(map[string]int)

func normalizeUserRatingKey(modelName string) string {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	
	vendorPrefixes := []string{"google/", "qwen/", "z-ai/", "anthropic/", "openai/", "meta/", "mistral/", "cohere/", "azure/", "aws/", "alibaba/", "baidu/", "tencent/", "minimaxai/"}
	for _, prefix := range vendorPrefixes {
		if strings.HasPrefix(modelName, prefix) {
			modelName = strings.TrimPrefix(modelName, prefix)
			break
		}
	}
	
	return modelName
}

func loadRatings() {
	userRepo := repository.NewUserRatingRepo()
	if userRatingsMap, err := userRepo.GetAllAsMap(); err == nil {
		for k, v := range userRatingsMap {
			userRatings[k] = v
		}
		log.Printf("[loadRatings] Loaded %d user ratings", len(userRatingsMap))
	} else {
		log.Printf("[loadRatings] Failed to load user ratings: %v", err)
	}

	sampleRepo := repository.NewSampleRatingRepo()
	if sampleRatingsMap, err := sampleRepo.GetAllAsMap(); err == nil {
		for k, v := range sampleRatingsMap {
			sampleRatings[k] = v.Score
		}
	}
}

func (d *Dispatcher) Dispatch(token *model.Token, requestBody []byte) ([]byte, int, error) {
	startTime := time.Now()

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	var modelItem *model.Model
	var err error

	modelName, _ := req["model"].(string)

	if modelName == "AUTO" || modelName == "POLL_ALL" {
		modelItem, err = d.modelService.GetNextModelAny()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models")
		}
	} else if modelName == "auto" || modelName == "Auto" {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	} else if modelName != "" {
		modelItem, err = d.modelService.GetByName(modelName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("model not found: %s", modelName)
		}
	}

	if modelItem == nil {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	}

	selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || selectedChannel == nil {
		return nil, 0, fmt.Errorf("channel not found for model")
	}

	if err := d.checkRateLimit(selectedChannel, 0); err != nil {
		return nil, 429, fmt.Errorf("channel rate limit exceeded: %w", err)
	}

	if err := d.checkModelRateLimit(modelItem, 0); err != nil {
		return nil, 429, fmt.Errorf("model rate limit exceeded: %w", err)
	}

	url := strings.TrimSuffix(selectedChannel.BaseURL, "/") + "/chat/completions"

	req["model"] = modelItem.Name
	modifiedBody, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	proxyReq, err := http.NewRequest("POST", url, bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+selectedChannel.APIKey)
	proxyReq.Header.Set("User-Agent", "AI-Gateway/1.0")

	resp, err := d.client.Do(proxyReq)
	if err != nil {
		d.logCall(token, selectedChannel, modelItem, startTime, 0, 503, err.Error())
		return nil, 503, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyStrLower := strings.ToLower(bodyStr)
	if strings.HasPrefix(bodyStrLower, "<!doctype") || strings.HasPrefix(bodyStrLower, "<html") {
		return nil, resp.StatusCode, fmt.Errorf("upstream API returned HTML error page (invalid API key or endpoint)")
	}

	var tokenUsed int
	isStream, _ := req["stream"].(bool)
	if strings.HasPrefix(bodyStr, "data: ") {
		isStream = true
		log.Printf("detected SSE response despite stream flag=%v, model=%s", req["stream"], modelItem.Name)
	}
	if !isStream {
		var relayResp RelayResponse
		if err := json.Unmarshal(body, &relayResp); err != nil {
			bodyLen := len(body)
			if bodyLen > 200 {
				bodyLen = 200
			}
			log.Printf("failed to unmarshal response for token usage: err=%v, body=%s, model=%s", err, string(body[:bodyLen]), modelItem.Name)
		} else {
			tokenUsed = relayResp.Usage.TotalTokens
		}
	}

	d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "")

	if resp.StatusCode == 200 {
		go d.updateChannelUsage(selectedChannel.ID, tokenUsed, modelItem.CostPerToken, modelItem.Currency)
		go d.updateModelUsage(modelItem.ID, tokenUsed)
		go d.extraRatingService.ApplyPenaltyAndReward(
			NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name),
		)
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		go d.saveSampleAsync(modelKey, string(requestBody), string(body), tokenUsed)
	}

	return body, resp.StatusCode, nil
}

func (d *Dispatcher) DispatchStream(token *model.Token, requestBody []byte) ([]byte, int, error) {
	startTime := time.Now()

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	var modelItem *model.Model
	var err error

	modelName, _ := req["model"].(string)

	if modelName == "AUTO" || modelName == "POLL_ALL" {
		modelItem, err = d.modelService.GetNextModelAny()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models")
		}
	} else if modelName == "auto" || modelName == "Auto" {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	} else if modelName != "" {
		modelItem, err = d.modelService.GetByName(modelName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("model not found: %s", modelName)
		}
	}

	if modelItem == nil {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	}

	selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || selectedChannel == nil {
		return nil, 0, fmt.Errorf("channel not found for model")
	}

	if err := d.checkRateLimit(selectedChannel, 0); err != nil {
		return nil, 429, fmt.Errorf("channel rate limit exceeded: %w", err)
	}

	if err := d.checkModelRateLimit(modelItem, 0); err != nil {
		return nil, 429, fmt.Errorf("model rate limit exceeded: %w", err)
	}

	url := strings.TrimSuffix(selectedChannel.BaseURL, "/") + "/chat/completions"

	req["model"] = modelItem.Name
	modifiedBody, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	proxyReq, err := http.NewRequest("POST", url, bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+selectedChannel.APIKey)
	proxyReq.Header.Set("User-Agent", "AI-Gateway/1.0")

	resp, err := d.client.Do(proxyReq)
	if err != nil {
		d.logCall(token, selectedChannel, modelItem, startTime, 0, 503, err.Error())
		return nil, 503, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	bodyStr := strings.TrimSpace(string(body))
	bodyStrLower := strings.ToLower(bodyStr)
	if strings.HasPrefix(bodyStrLower, "<!doctype") || strings.HasPrefix(bodyStrLower, "<html") {
		return nil, resp.StatusCode, fmt.Errorf("upstream API returned HTML error page (invalid API key or endpoint)")
	}

	tokenUsed := parseStreamUsageFromBytes(body)
	if tokenUsed == 0 {
		log.Printf("warning: streaming response returned 0 tokens for model=%s, channel=%s. NVIDIA streaming API does not provide usage info in streaming chunks.", modelItem.Name, selectedChannel.Name)
	}
	d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "")

	if resp.StatusCode == 200 {
		go d.updateChannelUsage(selectedChannel.ID, tokenUsed, modelItem.CostPerToken, modelItem.Currency)
		go d.updateModelUsage(modelItem.ID, tokenUsed)
		go d.extraRatingService.ApplyPenaltyAndReward(
			NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name),
		)
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		go d.saveSampleAsync(modelKey, string(requestBody), string(body), tokenUsed)
	}

	return body, resp.StatusCode, nil
}

func parseStreamUsageFromBytes(body []byte) int {
	var totalTokens int
	var responseText strings.Builder
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimLeft(data, " ")
			if data == "[DONE]" {
				break
			}
			if !strings.HasPrefix(data, "{") {
				continue
			}
			var chunk struct {
				Usage struct {
					TotalTokens int `json:"total_tokens"`
				} `json:"usage"`
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				dataLen := len(data)
				if dataLen > 100 {
					dataLen = 100
				}
				log.Printf("failed to unmarshal stream chunk for token usage: err=%v, data=%s", err, data[:dataLen])
			} else {
				if chunk.Usage.TotalTokens > 0 {
					totalTokens = chunk.Usage.TotalTokens
				}
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					responseText.WriteString(chunk.Choices[0].Delta.Content)
				}
			}
		}
	}
	if totalTokens == 0 && responseText.Len() > 0 {
		totalTokens = countTokensWithTiktoken(responseText.String())
	}
	return totalTokens
}

func countTokensWithTiktoken(text string) int {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("failed to get tiktoken encoding: %v", err)
		return 0
	}
	tokens := encoding.Encode(text, nil, nil)
	return len(tokens)
}

func (d *Dispatcher) logCall(token *model.Token, channel *model.Channel, modelItem *model.Model, startTime time.Time, tokenUsed int, status int, errMsg string) {
	latency := int(time.Since(startTime).Milliseconds())

	callLog := &model.CallLog{
		TokenName:    token.Name,
		ChannelName:  channel.Name,
		ModelName:    modelItem.Name,
		LatencyMs:    latency,
		TokenUsed:    tokenUsed,
		Status:       status,
		Error:        errMsg,
		CreatedAt:    time.Now(),
	}

	if err := d.logRepo.Save(callLog); err != nil {
		log.Printf("failed to save call log: %v", err)
	}
}

func (d *Dispatcher) saveSampleAsync(modelKey, requestContent, responseContent string, tokenCount int) {
	if err := d.sampleRepo.SaveSample(modelKey, requestContent, responseContent, tokenCount); err != nil {
		log.Printf("failed to save sample: %v", err)
	}
}

func (d *Dispatcher) checkRateLimit(channel *model.Channel, tokenUsed int) error {
	now := time.Now()

	if channel.ExpiresAt != nil && now.After(*channel.ExpiresAt) {
		return fmt.Errorf("channel expired at %s", channel.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if channel.TotalTokenLimit > 0 && channel.TotalTokens >= channel.TotalTokenLimit {
		return fmt.Errorf("total token limit exceeded: %d/%d", channel.TotalTokens, channel.TotalTokenLimit)
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for channel %s: %v", channel.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.rateLimitRepo.UpsertUsage(channel.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			return fmt.Errorf("calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "billing" && currentCount >= rule.MaxCount {
			return fmt.Errorf("billing limit exceeded: %d/%s per %s (quota exhausted)", currentCount/100, rule.Currency, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) getWindowDuration(window string) time.Duration {
	switch window {
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	case "week":
		return 7 * 24 * time.Hour
	case "month":
		return 30 * 24 * time.Hour
	case "quarter":
		return 90 * 24 * time.Hour
	case "year":
		return 365 * 24 * time.Hour
	default:
		return 0
	}
}

func (d *Dispatcher) getExchangeRate() float64 {
	config, err := d.systemConfigRepo.Get()
	if err != nil || config == nil {
		return 7.25
	}
	return config.ExchangeRate
}

func (d *Dispatcher) calculateCostInTargetCurrency(tokenUsed int, costPerToken float64, costCurrency string, targetCurrency string) int64 {
	if tokenUsed <= 0 || costPerToken <= 0 {
		return 0
	}
	baseCost := float64(tokenUsed) * costPerToken
	if costCurrency == targetCurrency {
		return int64(baseCost * 100)
	}
	exchangeRate := d.getExchangeRate()
	if costCurrency == "USD" && targetCurrency == "CNY" {
		return int64(baseCost * exchangeRate * 100)
	}
	if costCurrency == "CNY" && targetCurrency == "USD" {
		return int64(baseCost / exchangeRate * 100)
	}
	return int64(baseCost * 100)
}

func (d *Dispatcher) updateChannelUsage(channelID int64, tokenUsed int, costPerToken float64, costCurrency string) error {
	if err := d.channelRepo.IncrementUsage(channelID, tokenUsed); err != nil {
		log.Printf("failed to increment channel usage: %v", err)
	}

	channel, err := d.channelService.GetByID(channelID)
	if err != nil || channel == nil {
		return err
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for channel %s: %v", channel.Name, err)
		return nil
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channelID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		} else if rule.Type == "billing" {
			increment = d.calculateCostInTargetCurrency(tokenUsed, costPerToken, costCurrency, rule.Currency)
		}

		if usage == nil {
			windowStart = now
			if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				} else if rule.Type == "billing" {
					increment = d.calculateCostInTargetCurrency(tokenUsed, costPerToken, costCurrency, rule.Currency)
				}
				if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			} else {
				if err := d.rateLimitRepo.UpsertUsage(channelID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) checkModelRateLimit(modelItem *model.Model, tokenUsed int) error {
	if modelItem == nil {
		return nil
	}

	now := time.Now()

	if modelItem.ExpiresAt != nil && now.After(*modelItem.ExpiresAt) {
		return fmt.Errorf("model expired at %s", modelItem.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if modelItem.TotalTokenLimit > 0 && modelItem.TotalTokens >= modelItem.TotalTokenLimit {
		return fmt.Errorf("model total token limit exceeded: %d/%d", modelItem.TotalTokens, modelItem.TotalTokenLimit)
	}

	hasModelRules := modelItem.RateLimits != "" && modelItem.RateLimits != "[]"

	if !hasModelRules {
		return d.checkInheritedChannelLimits(modelItem, now)
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(modelItem.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for model %s: %v", modelItem.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.modelRateLimitRepo.GetUsage(modelItem.ID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.modelRateLimitRepo.UpsertUsage(modelItem.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			return fmt.Errorf("model calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("model tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) checkInheritedChannelLimits(modelItem *model.Model, now time.Time) error {
	channel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || channel == nil {
		return nil
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse inherited rate limits from channel %s for model %s: %v", channel.Name, modelItem.Name, err)
		return nil
	}

	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get inherited rate limit usage from channel %s rule %d for model %s: %v", channel.Name, idx, modelItem.Name, err)
			continue
		}

		var currentCount int64
		var windowStart time.Time

		if usage == nil {
			currentCount = 0
			windowStart = now
		} else {
			currentCount = usage.CurrentCount
			windowStart = usage.WindowStart

			if now.Sub(windowStart) >= windowDuration {
				currentCount = 0
				windowStart = now
				d.rateLimitRepo.UpsertUsage(channel.ID, idx, 0, windowStart, true)
			}
		}

		if rule.Type == "calls" && currentCount >= rule.MaxCount {
			return fmt.Errorf("inherited channel calls rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "tokens" && currentCount >= rule.MaxCount {
			return fmt.Errorf("inherited channel tokens rate limit exceeded: %d/%d per %s", currentCount, rule.MaxCount, rule.Window)
		}

		if rule.Type == "billing" && currentCount >= rule.MaxCount {
			return fmt.Errorf("inherited channel billing limit exceeded: %d/%s per %s (quota exhausted)", currentCount/100, rule.Currency, rule.Window)
		}
	}

	return nil
}

func (d *Dispatcher) updateModelUsage(modelID int64, tokenUsed int) error {
	if err := d.modelRepo.IncrementUsage(modelID, tokenUsed); err != nil {
		log.Printf("failed to increment model usage: %v", err)
	}

	modelItem, err := d.modelRepo.GetByID(modelID)
	if err != nil || modelItem == nil {
		return err
	}

	hasModelRules := modelItem.RateLimits != "" && modelItem.RateLimits != "[]"

	if !hasModelRules {
		d.updateInheritedChannelUsage(modelItem, tokenUsed)
		return nil
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(modelItem.RateLimits), &rules); err != nil {
		log.Printf("failed to parse rate limits for model %s: %v", modelItem.Name, err)
		return nil
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.modelRateLimitRepo.GetUsage(modelID, idx)
		if err != nil {
			log.Printf("failed to get rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		}

		if usage == nil {
			windowStart = now
			if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				}
				if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
				}
			} else {
				if err := d.modelRateLimitRepo.UpsertUsage(modelID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert rate limit usage for model %s rule %d: %v", modelItem.Name, idx, err)
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) updateInheritedChannelUsage(modelItem *model.Model, tokenUsed int) {
	channel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || channel == nil {
		return
	}

	if channel.RateLimits == "" || channel.RateLimits == "[]" {
		return
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(channel.RateLimits), &rules); err != nil {
		log.Printf("failed to parse inherited rate limits from channel %s for model %s: %v", channel.Name, modelItem.Name, err)
		return
	}

	now := time.Now()
	for idx, rule := range rules {
		windowDuration := d.getWindowDuration(rule.Window)
		if windowDuration == 0 {
			continue
		}

		usage, err := d.rateLimitRepo.GetUsage(channel.ID, idx)
		if err != nil {
			log.Printf("failed to get inherited rate limit usage from channel %s rule %d for model %s: %v", channel.Name, idx, modelItem.Name, err)
			continue
		}

		var windowStart time.Time
		var increment int64 = 1

		if rule.Type == "tokens" {
			increment = int64(tokenUsed)
		} else if rule.Type == "billing" {
			increment = d.calculateCostInTargetCurrency(tokenUsed, modelItem.CostPerToken, modelItem.Currency, rule.Currency)
		}

		if usage == nil {
			windowStart = now
			if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, true); err != nil {
				log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
			}
		} else {
			windowStart = usage.WindowStart
			if now.Sub(windowStart) >= windowDuration {
				windowStart = now
				increment = int64(tokenUsed)
				if rule.Type == "calls" {
					increment = 1
				} else if rule.Type == "billing" {
					increment = d.calculateCostInTargetCurrency(tokenUsed, modelItem.CostPerToken, modelItem.Currency, rule.Currency)
				}
				if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, true); err != nil {
					log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			} else {
				if err := d.rateLimitRepo.UpsertUsage(channel.ID, idx, increment, windowStart, false); err != nil {
					log.Printf("failed to upsert inherited rate limit usage for channel %s rule %d: %v", channel.Name, idx, err)
				}
			}
		}
	}
}

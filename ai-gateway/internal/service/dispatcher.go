package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
)

type Dispatcher struct {
	channelService     *ChannelService
	modelService       *ModelService
	logRepo            *repository.LogRepo
	sampleRepo         *repository.SampleRepo
	channelRepo        *repository.ChannelRepo
	rateLimitRepo      *repository.RateLimitRepo
	modelRateLimitRepo *repository.ModelRateLimitRepo
	tokenRateLimitRepo *repository.TokenRateLimitRepo
	modelRepo          *repository.ModelRepo
	systemConfigRepo   *repository.SystemConfigRepo
	extraRatingService *ExtraRatingService
	tokenRepo          *repository.TokenRepo
	client             *http.Client
}

func NewDispatcher() *Dispatcher {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Dispatcher{
		channelService:     NewChannelService(),
		modelService:       NewModelService(),
		logRepo:            repository.NewLogRepo(),
		sampleRepo:         repository.NewSampleRepo(),
		channelRepo:        repository.NewChannelRepo(),
		rateLimitRepo:      repository.NewRateLimitRepo(),
		modelRateLimitRepo: repository.NewModelRateLimitRepo(),
		tokenRateLimitRepo: repository.NewTokenRateLimitRepo(),
		modelRepo:          repository.NewModelRepo(),
		systemConfigRepo:   repository.NewSystemConfigRepo(),
		extraRatingService: NewExtraRatingService(),
		tokenRepo:          repository.NewTokenRepo(),
		client: &http.Client{
			Timeout:   1200 * time.Second,
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

type StreamResponse struct {
	Resp        *http.Response
	ChannelName string
	ModelName   string
	ChannelID   int64
	ModelID     int64
	TokenUsed   int
}

func (d *Dispatcher) ListEnabledModels() ([]*model.Model, error) {
	return d.modelService.ListEnabled()
}

func (d *Dispatcher) GetNextModelSmart(format, modelType string) (*model.Model, error) {
	models, err := d.GetRankedModelsSmart(format, modelType, 1)
	if err != nil || len(models) == 0 {
		return d.modelService.GetNextModelGlobal(format, modelType)
	}
	return models[0], nil
}

func (d *Dispatcher) GetRankedModelsSmart(format, modelType string, limit int) ([]*model.Model, error) {
	config, err := d.systemConfigRepo.Get()
	if err != nil {
		return nil, err
	}

	dispatchMode := "polling"
	if config != nil && config.DispatchMode != "" {
		dispatchMode = config.DispatchMode
	}

	if dispatchMode != "smart" {
		selectedModel, err := d.modelService.GetNextModelGlobal(format, modelType)
		if err != nil || selectedModel == nil {
			return nil, err
		}
		return []*model.Model{selectedModel}, nil
	}

	modelRatingSvc := NewModelRatingService()
	allScores, err := modelRatingSvc.CalculateAllScores()
	if err != nil {
		log.Printf("[GetRankedModelsSmart] failed to calculate scores: %v", err)
		return nil, err
	}

	var rankedModels []*model.Model
	for _, score := range allScores {
		if format != "" && score.Format != format {
			continue
		}
		if modelType != "" && score.ModelType != modelType {
			continue
		}

		selectedModel, err := d.modelRepo.GetByChannelNameAndModel(score.ChannelName, score.ModelName)
		if err != nil || selectedModel == nil {
			continue
		}

		rankedModels = append(rankedModels, selectedModel)
		if limit > 0 && len(rankedModels) >= limit {
			break
		}
	}

	if len(rankedModels) == 0 {
		return nil, fmt.Errorf("no available models for format=%s type=%s", format, modelType)
	}

	return rankedModels, nil
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
		LatencyWeight:      repoWeights.LatencyWeight,
		ReliabilityWeight:  repoWeights.ReliabilityWeight,
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
	if ur := getUserRating(normalizedKey); ur != 0 {
		userRating = ur
	} else if ur := getUserRating(m.Name); ur != 0 {
		userRating = ur
	} else if ur := getUserRating(strings.ToLower(m.Name)); ur != 0 {
		userRating = ur
	}
	userScore := float64(userRating) / 100.0

	sampleRating := 50
	sampleKey := NormalizeModelKey(m.ChannelName, m.Format, m.Type, m.Name)
	if sr := getSampleRating(sampleKey); sr != 0 {
		sampleRating = sr
	}
	sampleScore := float64(sampleRating) / 100.0

	penalty, reward, _ := d.extraRatingService.GetModelExtraScore(sampleKey)
	extraScore := float64(reward + penalty)

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

// selectModelAndChannel 提取模型选择和渠道获取的公共逻辑
// 参数:
//   - token: 请求token
//   - req: 解析后的请求体
// 返回: modelItem, selectedChannel, error
func (d *Dispatcher) selectModelAndChannel(token *model.Token, req map[string]interface{}) (*model.Model, *model.Channel, error) {
	var modelItem *model.Model
	var err error

	modelName, _ := req["model"].(string)

	if modelName == "POLL_ALL" {
		modelItem, err = d.modelService.GetNextModelAny()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, nil, fmt.Errorf("no available models")
		}
	} else if modelName == "AUTO" || modelName == "__AUTO__" {
		config, _ := d.systemConfigRepo.Get()
		dispatchMode := "polling"
		if config != nil {
			dispatchMode = config.DispatchMode
		}
		if dispatchMode == "smart" {
			modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		} else {
			modelItem, err = d.modelService.GetNextModelAny()
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, nil, fmt.Errorf("no available models")
		}
	} else if modelName == "auto" || modelName == "Auto" {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, nil, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	} else if modelName != "" {
		modelItem, err = d.modelService.GetByName(modelName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get model: %w", err)
		}
		if modelItem == nil {
			return nil, nil, fmt.Errorf("model not found: %s", modelName)
		}
	}

	if modelItem == nil {
		modelItem, err = d.GetNextModelSmart(token.Format, token.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get next model: %w", err)
		}
		if modelItem == nil {
			return nil, nil, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
		}
	}

	selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || selectedChannel == nil {
		return nil, nil, fmt.Errorf("channel not found for model")
	}

	if err := d.checkRateLimit(selectedChannel, 0); err != nil {
		return nil, nil, fmt.Errorf("channel rate limit exceeded: %w", err)
	}

	if err := d.checkModelRateLimit(modelItem, 0); err != nil {
		return nil, nil, fmt.Errorf("model rate limit exceeded: %w", err)
	}

	return modelItem, selectedChannel, nil
}

func (d *Dispatcher) Dispatch(token *model.Token, requestBody []byte) ([]byte, int, error) {
	return d.dispatch(token, requestBody, false)
}

func (d *Dispatcher) DispatchStream(token *model.Token, requestBody []byte) ([]byte, int, error) {
	return d.dispatch(token, requestBody, true)
}

func (d *Dispatcher) DispatchStreamDirect(token *model.Token, requestBody []byte) (*StreamResponse, int, error) {
	startTime := time.Now()

	if err := d.checkTokenRateLimit(token); err != nil {
		return nil, 0, fmt.Errorf("token rate limit: %w", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	modelName, _ := req["model"].(string)

	useSmartRetry := modelName == "auto" || modelName == "AUTO" || modelName == "__AUTO__" || modelName == "Auto" || modelName == "POLL_ALL"

	var rankedModels []*model.Model
	var err error

	if useSmartRetry {
		rankedModels, err = d.GetRankedModelsSmart(token.Format, token.Type, 3)
		if err != nil {
			return nil, 0, err
		}
	} else if modelName != "" {
		singleModel, err := d.modelService.GetByName(modelName)
		if err != nil {
			return nil, 0, err
		}
		if singleModel == nil {
			return nil, 0, fmt.Errorf("model not found: %s", modelName)
		}
		rankedModels = []*model.Model{singleModel}
	} else {
		rankedModels, err = d.GetRankedModelsSmart(token.Format, token.Type, 3)
		if err != nil {
			return nil, 0, err
		}
	}

	var lastErr error
	var lastStatusCode int
	var lastBody []byte

	for i, modelItem := range rankedModels {
		selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
		if err != nil || selectedChannel == nil {
			lastErr = fmt.Errorf("channel not found for model")
			continue
		}

		if err := d.checkRateLimit(selectedChannel, 0); err != nil {
			lastErr = fmt.Errorf("channel rate limit exceeded: %w", err)
			continue
		}

		if err := d.checkModelRateLimit(modelItem, 0); err != nil {
			lastErr = fmt.Errorf("model rate limit exceeded: %w", err)
			continue
		}

		url := strings.TrimSuffix(selectedChannel.BaseURL, "/") + "/chat/completions"

		req["model"] = modelItem.Name
		modifiedBody, err := json.Marshal(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal request: %w", err)
			continue
		}

		proxyReq, err := http.NewRequest("POST", url, bytes.NewReader(modifiedBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+selectedChannel.APIKey)
		proxyReq.Header.Set("User-Agent", "ScoreRoute/1.0")

		resp, err := d.client.Do(proxyReq)
		if err != nil {
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
				log.Printf("[DispatchStreamDirect] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
			d.logCall(token, selectedChannel, modelItem, startTime, 0, 503, err.Error())
			lastErr = fmt.Errorf("upstream request failed: %w", err)
			lastStatusCode = 503
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
				log.Printf("[DispatchStreamDirect] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
			lastErr = fmt.Errorf("failed to read response: %w", err)
			lastStatusCode = resp.StatusCode
			continue
		}

		bodyStr := strings.TrimSpace(string(body))
		bodyStrLower := strings.ToLower(bodyStr)
		if strings.HasPrefix(bodyStrLower, "<!doctype") || strings.HasPrefix(bodyStrLower, "<html") {
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
				log.Printf("[DispatchStreamDirect] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
			lastErr = fmt.Errorf("upstream API returned HTML error page (invalid API key or endpoint)")
			lastStatusCode = resp.StatusCode
			lastBody = body
			continue
		}

		if resp.StatusCode == 200 {
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
				log.Printf("[DispatchStreamDirect] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
			log.Printf("[DispatchStreamDirect] succeeded on try %d: %s/%s score rank %d", i+1, selectedChannel.Name, modelItem.Name, i+1)

			reader := bytes.NewReader(body)
			return &StreamResponse{
				Resp: &http.Response{
					StatusCode:    resp.StatusCode,
					Body:          io.NopCloser(reader),
					ContentLength: int64(len(body)),
					Header:        resp.Header,
				},
				ChannelName: selectedChannel.Name,
				ModelName:   modelItem.Name,
				ChannelID:   selectedChannel.ID,
				ModelID:     modelItem.ID,
			}, resp.StatusCode, nil
		}

		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
			log.Printf("[DispatchStreamDirect] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
		}
		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		lastStatusCode = resp.StatusCode
		lastBody = body
		log.Printf("[DispatchStreamDirect] failed on try %d: %s/%s status=%d, trying next model", i+1, selectedChannel.Name, modelItem.Name, resp.StatusCode)
	}

	if lastBody != nil {
		return &StreamResponse{
			Resp: &http.Response{
				StatusCode:    lastStatusCode,
				Body:          io.NopCloser(bytes.NewReader(lastBody)),
				ContentLength: int64(len(lastBody)),
				Header:        make(http.Header),
			},
			ChannelName: "",
			ModelName:   "",
		}, lastStatusCode, lastErr
	}
	return nil, lastStatusCode, lastErr
}

func (d *Dispatcher) LogStreamCompletion(tokenID int64, tokenName string, channelName string, modelName string, statusCode int, latency int, tokenUsed int) {
	if statusCode != 200 {
		d.logRepo.Create(&model.CallLog{
			TokenName:   tokenName,
			ChannelName: channelName,
			ModelName:   modelName,
			LatencyMs:   latency,
			TokenUsed:   0,
			Status:      statusCode,
			Error:       "stream failed",
		})
		return
	}

	d.logRepo.Create(&model.CallLog{
		TokenName:   tokenName,
		ChannelName: channelName,
		ModelName:   modelName,
		LatencyMs:   latency,
		TokenUsed:   tokenUsed,
		Status:      statusCode,
	})

	if tokenUsed > 0 {
		token, err := d.tokenRepo.GetByID(tokenID)
		if err == nil && token != nil {
			d.updateTokenUsage(token, tokenUsed)
		}
	}
}

func (d *Dispatcher) ParseStreamUsage(body []byte) int {
	return parseStreamUsageFromBytes(body)
}

func (d *Dispatcher) extractModelKeyFromResponse(body []byte) string {
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimLeft(data, " ")
			if !strings.HasPrefix(data, "{") {
				continue
			}
			var chunk struct {
				Model string `json:"model"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk.Model != "" {
					return chunk.Model
				}
			}
		}
	}
	return ""
}

func (d *Dispatcher) dispatch(token *model.Token, requestBody []byte, forceStream bool) ([]byte, int, error) {
	startTime := time.Now()

	if err := d.checkTokenRateLimit(token); err != nil {
		return nil, 0, fmt.Errorf("token rate limit: %w", err)
	}

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	modelName, _ := req["model"].(string)

	useSmartRetry := modelName == "auto" || modelName == "AUTO" || modelName == "__AUTO__" || modelName == "Auto" || modelName == "POLL_ALL"

	var rankedModels []*model.Model
	var err error

	if useSmartRetry {
		rankedModels, err = d.GetRankedModelsSmart(token.Format, token.Type, 3)
		if err != nil {
			return nil, 0, err
		}
	} else if modelName != "" {
		singleModel, err := d.modelService.GetByName(modelName)
		if err != nil {
			return nil, 0, err
		}
		if singleModel == nil {
			return nil, 0, fmt.Errorf("model not found: %s", modelName)
		}
		rankedModels = []*model.Model{singleModel}
	} else {
		rankedModels, err = d.GetRankedModelsSmart(token.Format, token.Type, 3)
		if err != nil {
			return nil, 0, err
		}
	}

	var lastErr error
	var lastStatusCode int
	var lastBody []byte

	for i, modelItem := range rankedModels {
		selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
		if err != nil || selectedChannel == nil {
			lastErr = fmt.Errorf("channel not found for model")
			continue
		}

		if err := d.checkRateLimit(selectedChannel, 0); err != nil {
			lastErr = fmt.Errorf("channel rate limit exceeded: %w", err)
			continue
		}

		if err := d.checkModelRateLimit(modelItem, 0); err != nil {
			lastErr = fmt.Errorf("model rate limit exceeded: %w", err)
			continue
		}

		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)

		url := strings.TrimSuffix(selectedChannel.BaseURL, "/") + "/chat/completions"

		req["model"] = modelItem.Name
		modifiedBody, err := json.Marshal(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal request: %w", err)
			continue
		}

		proxyReq, err := http.NewRequest("POST", url, bytes.NewReader(modifiedBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		proxyReq.Header.Set("Content-Type", "application/json")
		proxyReq.Header.Set("Authorization", "Bearer "+selectedChannel.APIKey)
		proxyReq.Header.Set("User-Agent", "ScoreRoute/1.0")

		resp, err := d.client.Do(proxyReq)
		if err != nil {
			d.logCall(token, selectedChannel, modelItem, startTime, 0, 503, err.Error())
			lastErr = fmt.Errorf("upstream request failed: %w", err)
			lastStatusCode = 503
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			lastStatusCode = resp.StatusCode
			continue
		}

		bodyStr := strings.TrimSpace(string(body))
		bodyStrLower := strings.ToLower(bodyStr)
		if strings.HasPrefix(bodyStrLower, "<!doctype") || strings.HasPrefix(bodyStrLower, "<html") {
			lastErr = fmt.Errorf("upstream API returned HTML error page (invalid API key or endpoint)")
			lastStatusCode = resp.StatusCode
			lastBody = body
			continue
		}

		isStream := forceStream
		if !isStream {
			isStream, _ = req["stream"].(bool)
		}
		if strings.HasPrefix(bodyStr, "data: ") {
			if !isStream {
				log.Printf("detected SSE response despite stream flag=%v, model=%s", req["stream"], modelItem.Name)
			}
			isStream = true
		}

		tokenUsed := d.calculateTokenUsage(body, isStream, modelItem.Name)

		d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "")

		if err := d.extraRatingService.ApplyPenaltyAndReward(modelKey); err != nil {
			log.Printf("[dispatch] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
		}

		if resp.StatusCode == 200 {
			d.performAsyncUpdates(selectedChannel, modelItem, tokenUsed)
			d.updateTokenUsage(token, tokenUsed)
		}

		if resp.StatusCode == 200 && tokenUsed > 0 {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := d.saveSampleAsyncContext(ctx, modelKey, string(requestBody), string(body), tokenUsed); err != nil {
					log.Printf("[ERROR] saveSampleAsync failed: model=%s, err=%v", modelItem.Name, err)
				}
			}()
		}

		if resp.StatusCode == 200 {
			log.Printf("[dispatch] succeeded on try %d: %s/%s score rank %d", i+1, selectedChannel.Name, modelItem.Name, i+1)
			return body, resp.StatusCode, nil
		}

		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		lastStatusCode = resp.StatusCode
		lastBody = body
		log.Printf("[dispatch] failed on try %d: %s/%s status=%d, trying next model", i+1, selectedChannel.Name, modelItem.Name, resp.StatusCode)
	}

	if lastBody != nil {
		return lastBody, lastStatusCode, lastErr
	}
	return nil, lastStatusCode, lastErr
}

func (d *Dispatcher) calculateTokenUsage(body []byte, isStream bool, modelName string) int {
	if isStream {
		return parseStreamUsageFromBytes(body)
	}

	var relayResp RelayResponse
	if err := json.Unmarshal(body, &relayResp); err != nil {
		bodyLen := len(body)
		if bodyLen > 200 {
			bodyLen = 200
		}
		log.Printf("failed to unmarshal response for token usage: err=%v, body=%s, model=%s", err, string(body[:bodyLen]), modelName)
		return 0
	}
	return relayResp.Usage.TotalTokens
}

func (d *Dispatcher) performAsyncUpdates(selectedChannel *model.Channel, modelItem *model.Model, tokenUsed int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := d.updateChannelUsageContext(ctx, selectedChannel.ID, tokenUsed, modelItem.CostPerToken, modelItem.Currency); err != nil {
			log.Printf("[ERROR] updateChannelUsage failed: channel=%s, err=%v", selectedChannel.Name, err)
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := d.updateModelUsageContext(ctx, modelItem.ID, tokenUsed); err != nil {
			log.Printf("[ERROR] updateModelUsage failed: model=%s, err=%v", modelItem.Name, err)
		}
	}()
}

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
	channelService      *ChannelService
	modelService        *ModelService
	logRepo             *repository.LogRepo
	sampleRepo          *repository.SampleRepo
	channelRepo         *repository.ChannelRepo
	rateLimitRepo       *repository.RateLimitRepo
	modelRateLimitRepo  *repository.ModelRateLimitRepo
	modelRepo           *repository.ModelRepo
	systemConfigRepo    *repository.SystemConfigRepo
	extraRatingService  *ExtraRatingService
	client              *http.Client
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

	modelRatingSvc := NewModelRatingService()
	allScores, err := modelRatingSvc.CalculateAllScores()
	if err != nil {
		log.Printf("[GetNextModelSmart] failed to calculate scores: %v", err)
		return d.modelService.GetNextModelGlobal(format, modelType)
	}

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

		d.modelRepo.IncrementCallCount(selectedModel.ID)
		d.channelRepo.IncrementCallCount(selectedModel.ChannelID)

		log.Printf("[GetNextModelSmart] selected %s/%s score=%.2f rank=%d",
			score.ChannelName, score.ModelName, score.Score, score.Rank)
		return selectedModel, nil
	}

	return d.modelService.GetNextModelGlobal(format, modelType)
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
	extraScore := float64(reward+penalty)

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

// TODO [代码质量-已知问题]: Dispatch 和 DispatchStream 有约95%重复代码
// 问题: 两个函数几乎完全相同，修改一个可能忘记修改另一个
// 风险: 如果需要修改模型选择/rate limit等逻辑，必须同时修改两处
// 建议: 未来重构时提取公共逻辑到 doDispatch() 方法
// 状态: 重构中 (2026-04-15)
// 重构策略: 提取公共逻辑到 selectModelAndChannel() 方法

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
	startTime := time.Now()

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	modelItem, selectedChannel, err := d.selectModelAndChannel(token, req)
	if err != nil {
		return nil, 0, err
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
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndRewardContext(ctx, modelKey); err != nil {
				log.Printf("[ERROR] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := d.saveSampleAsyncContext(ctx, modelKey, string(requestBody), string(body), tokenUsed); err != nil {
				log.Printf("[ERROR] saveSampleAsync failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	return body, resp.StatusCode, nil
}

// TODO [代码质量-已知问题]: Dispatch 和 DispatchStream 有约95%重复代码
// 问题: 两个函数几乎完全相同，修改一个可能忘记修改另一个
// 风险: 如果需要修改模型选择/rate limit等逻辑，必须同时修改两处
// 建议: 未来重构时提取公共逻辑到 doDispatch() 方法
// 状态: 暂不重构 (2026-04-15) - 重构风险高，功能正常
func (d *Dispatcher) DispatchStream(token *model.Token, requestBody []byte) ([]byte, int, error) {
	startTime := time.Now()

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, 0, fmt.Errorf("invalid request body: %w", err)
	}

	modelItem, selectedChannel, err := d.selectModelAndChannel(token, req)
	if err != nil {
		return nil, 0, err
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
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndRewardContext(ctx, modelKey); err != nil {
				log.Printf("[ERROR] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := d.saveSampleAsyncContext(ctx, modelKey, string(requestBody), string(body), tokenUsed); err != nil {
				log.Printf("[ERROR] saveSampleAsync failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	return body, resp.StatusCode, nil
}



func (d *Dispatcher) DispatchStreamToWriter(w io.Writer, token *model.Token, requestBody []byte) (int, error) {
	startTime := time.Now()

	var req map[string]interface{}
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return 0, fmt.Errorf("invalid request body: %w", err)
	}

	modelItem, selectedChannel, err := d.selectModelAndChannel(token, req)
	if err != nil {
		return 0, err
	}

	url := strings.TrimSuffix(selectedChannel.BaseURL, "/") + "/chat/completions"

	req["model"] = modelItem.Name
	modifiedBody, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	proxyReq, err := http.NewRequest("POST", url, bytes.NewReader(modifiedBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+selectedChannel.APIKey)
	proxyReq.Header.Set("User-Agent", "AI-Gateway/1.0")

	resp, err := d.client.Do(proxyReq)
	if err != nil {
		d.logCall(token, selectedChannel, modelItem, startTime, 0, 503, err.Error())
		return 503, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBuf := &bytes.Buffer{}
	writer := io.MultiWriter(w, bodyBuf)

	tokenUsed := 0
	lastKeepalive := time.Now()
	keepaliveInterval := 30 * time.Second

	for {
		buf := make([]byte, 4096)
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := writer.Write(buf[:n]); err != nil {
				log.Printf("[WARN] client disconnected: %v", err)
				d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "client disconnected")
				return resp.StatusCode, nil
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}

		if time.Since(lastKeepalive) >= keepaliveInterval {
			fmt.Fprintf(w, ": keepalive\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			lastKeepalive = time.Now()
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, fmt.Sprintf("read error: %v", err))
			return resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
		}
	}

	bodyStr := strings.TrimSpace(bodyBuf.String())
	bodyStrLower := strings.ToLower(bodyStr)
	if strings.HasPrefix(bodyStrLower, "<!doctype") || strings.HasPrefix(bodyStrLower, "<html") {
		d.logCall(token, selectedChannel, modelItem, startTime, 0, resp.StatusCode, "upstream API returned HTML error page")
		return resp.StatusCode, fmt.Errorf("upstream API returned HTML error page")
	}

	tokenUsed = parseStreamUsageFromBytes(bodyBuf.Bytes())
	if tokenUsed == 0 {
		log.Printf("warning: streaming response returned 0 tokens for model=%s, channel=%s", modelItem.Name, selectedChannel.Name)
	}

	d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "")

	if resp.StatusCode == 200 {
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
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
			if err := d.extraRatingService.ApplyPenaltyAndRewardContext(ctx, modelKey); err != nil {
				log.Printf("[ERROR] ApplyPenaltyAndReward failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := NormalizeModelKey(selectedChannel.Name, selectedChannel.Format, modelItem.Type, modelItem.Name)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := d.saveSampleAsyncContext(ctx, modelKey, string(requestBody), bodyStr, tokenUsed); err != nil {
				log.Printf("[ERROR] saveSampleAsync failed: model=%s, err=%v", modelItem.Name, err)
			}
		}()
	}

	return resp.StatusCode, nil
}

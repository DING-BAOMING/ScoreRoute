package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"

	"github.com/pkoukk/tiktoken-go"
)

type Dispatcher struct {
	channelService *ChannelService
	modelService   *ModelService
	logRepo        *repository.LogRepo
	sampleRepo     *repository.SampleRepo
	channelRepo    *repository.ChannelRepo
	rateLimitRepo  *repository.RateLimitRepo
	client         *http.Client
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		channelService: NewChannelService(),
		modelService:   NewModelService(),
		logRepo:        repository.NewLogRepo(),
		sampleRepo:     repository.NewSampleRepo(),
		channelRepo:    repository.NewChannelRepo(),
		rateLimitRepo:  repository.NewRateLimitRepo(),
		client: &http.Client{
			Timeout: 300 * time.Second,
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
		modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
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
		if token.ModelName == "__POLL_ALL__" {
			modelItem, err = d.modelService.GetNextModelAny()
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models")
			}
		} else if token.ModelName == "__AUTO__" {
			modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
			}
		} else {
			modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
			}
		}
	}

	selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || selectedChannel == nil {
		return nil, 0, fmt.Errorf("channel not found for model")
	}

	if err := d.checkRateLimit(selectedChannel, 0); err != nil {
		return nil, 429, fmt.Errorf("rate limit exceeded: %w", err)
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

	var tokenUsed int
	bodyStr := string(body)
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
		go d.updateChannelUsage(selectedChannel.ID, tokenUsed)
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := selectedChannel.Format + "_" + modelItem.Type + "_" + modelItem.Name
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
		modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
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
		if token.ModelName == "__POLL_ALL__" {
			modelItem, err = d.modelService.GetNextModelAny()
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models")
			}
		} else if token.ModelName == "__AUTO__" {
			modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
			}
		} else {
			modelItem, err = d.modelService.GetNextModelGlobal(token.Format, token.Type)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to get next model: %w", err)
			}
			if modelItem == nil {
				return nil, 0, fmt.Errorf("no available models for format=%s type=%s", token.Format, token.Type)
			}
		}
	}

	selectedChannel, err := d.channelService.GetByID(modelItem.ChannelID)
	if err != nil || selectedChannel == nil {
		return nil, 0, fmt.Errorf("channel not found for model")
	}

	if err := d.checkRateLimit(selectedChannel, 0); err != nil {
		return nil, 429, fmt.Errorf("rate limit exceeded: %w", err)
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

	tokenUsed := parseStreamUsageFromBytes(body)
	if tokenUsed == 0 {
		log.Printf("warning: streaming response returned 0 tokens for model=%s, channel=%s. NVIDIA streaming API does not provide usage info in streaming chunks.", modelItem.Name, selectedChannel.Name)
	}
	d.logCall(token, selectedChannel, modelItem, startTime, tokenUsed, resp.StatusCode, "")

	if resp.StatusCode == 200 {
		go d.updateChannelUsage(selectedChannel.ID, tokenUsed)
	}

	if resp.StatusCode == 200 && tokenUsed > 0 {
		modelKey := selectedChannel.Format + "_" + modelItem.Type + "_" + modelItem.Name
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
	case "year":
		return 365 * 24 * time.Hour
	default:
		return 0
	}
}

func (d *Dispatcher) updateChannelUsage(channelID int64, tokenUsed int) error {
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

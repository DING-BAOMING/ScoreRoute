package service

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/repository"
	"github.com/pkoukk/tiktoken-go"
)

// parseStreamUsageFromBytes 从流式响应中解析 token 使用量
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

// countTokensWithTiktoken 使用 tiktoken 计算 token 数量
func countTokensWithTiktoken(text string) int {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		log.Printf("failed to get tiktoken encoding: %v", err)
		return 0
	}
	tokens := encoding.Encode(text, nil, nil)
	return len(tokens)
}

// normalizeUserRatingKey 标准化用户名用于用户评分
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

// getWindowDuration 根据窗口字符串返回对应时长
func getWindowDuration(window string) time.Duration {
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

var (
	userRatings   = make(map[string]int)
	sampleRatings = make(map[string]int)
	ratingsMu     sync.RWMutex
)

func getUserRating(key string) int {
	ratingsMu.RLock()
	defer ratingsMu.RUnlock()
	return userRatings[key]
}

func setUserRating(key string, val int) {
	ratingsMu.Lock()
	defer ratingsMu.Unlock()
	userRatings[key] = val
}

func getSampleRating(key string) int {
	ratingsMu.RLock()
	defer ratingsMu.RUnlock()
	return sampleRatings[key]
}

func setSampleRating(key string, val int) {
	ratingsMu.Lock()
	defer ratingsMu.Unlock()
	sampleRatings[key] = val
}

func loadRatings() {
	ratingsMu.Lock()
	defer ratingsMu.Unlock()

	userRepo := repository.NewUserRatingRepo()

	if ratings, err := userRepo.GetDeduplicatedUserRatings(); err == nil {
		for _, r := range ratings {
			modelName, _ := r["model_name"].(string)
			rating, _ := r["user_rating"].(int)
			userRatings[strings.ToLower(modelName)] = rating
		}
		log.Printf("[loadRatings] Loaded %d user ratings from deduplicated", len(ratings))
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

package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"ai-gateway/internal/service"
)

var (
	tokenSvc      *service.TokenService
	rateLimitRepo *repository.TokenRateLimitRepo
)

func init() {
	tokenSvc = service.NewTokenService()
	rateLimitRepo = repository.NewTokenRateLimitRepo()
}

func RateLimitHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			var apiKey string
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				apiKey = authHeader[7:]
			} else {
				apiKey = authHeader
			}
			token, err := tokenSvc.GetByKey(apiKey)
			if err == nil && token != nil {
				setRateLimitHeaders(c, token)
				c.Next()
				return
			}
		}
		c.Header("X-RateLimit-Limit", "1000")
		c.Header("X-RateLimit-Remaining", "1000")
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
		c.Next()
	}
}

func setRateLimitHeaders(c *gin.Context, token *model.Token) {
	c.Header("X-RateLimit-Limit", "1000")
	c.Header("X-RateLimit-Remaining", "999")
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))

	if token.RateLimits == "" || token.RateLimits == "[]" {
		return
	}

	var rules []model.RateLimitRule
	if err := json.Unmarshal([]byte(token.RateLimits), &rules); err != nil {
		return
	}

	for idx, rule := range rules {
		if rule.Type == "calls" || rule.Type == "tokens" {
			usage, err := rateLimitRepo.GetUsage(token.ID, idx)
			if err != nil {
				continue
			}

			var remaining int64
			var reset int64

			if usage == nil {
				remaining = rule.MaxCount
				reset = time.Now().Add(getWindowDuration(rule.Window)).Unix()
			} else {
				remaining = rule.MaxCount - usage.CurrentCount
				if remaining < 0 {
					remaining = 0
				}
				reset = usage.WindowStart.Add(getWindowDuration(rule.Window)).Unix()
			}

			prefix := "X-RateLimit-"
			if rule.Type == "tokens" {
				prefix = "X-RateLimit-Tokens-"
			}

			c.Header(prefix+"Limit", fmt.Sprintf("%d", rule.MaxCount))
			c.Header(prefix+"Remaining", fmt.Sprintf("%d", remaining))
			c.Header(prefix+"Reset", fmt.Sprintf("%d", reset))
			break
		}
	}
}

func getWindowDuration(window string) time.Duration {
	switch window {
	case "second":
		return time.Second
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

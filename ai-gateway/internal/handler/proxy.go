package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/repository"
	"ai-gateway/internal/service"
)

type ProxyHandler struct {
	dispatcher    *service.Dispatcher
	tokenSvc      *service.TokenService
	rateLimitRepo *repository.TokenRateLimitRepo
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		dispatcher:    service.NewDispatcher(),
		tokenSvc:      service.NewTokenService(),
		rateLimitRepo: repository.NewTokenRateLimitRepo(),
	}
}

func (h *ProxyHandler) Handle(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Missing authorization header"}})
		return
	}

	var apiKey string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		apiKey = authHeader[7:]
	} else {
		apiKey = authHeader
	}

	token, err := h.tokenSvc.GetByKey(apiKey)
	if err != nil || token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Invalid API key"}})
		return
	}

	if token.Enabled != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "API key is disabled"}})
		return
	}

	if token.AutoDisabled == 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": fmt.Sprintf("API key is auto-disabled: %s", token.AutoDisableReason)}})
		return
	}

	h.setRateLimitHeaders(c, token)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read request body"}})
		return
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(body, &reqBody); err == nil {
		if stream, ok := reqBody["stream"].(bool); ok && stream {
			h.HandleStreamWithBody(c, token, body)
			return
		}
	}

	respBody, statusCode, err := h.dispatcher.Dispatch(token, body)
	log.Printf("Dispatch result: statusCode=%d, err=%v, bodyLen=%d", statusCode, err, len(respBody))
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "rate limit") {
			h.setRateLimitHeaders(c, token)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": errMsg}})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": errMsg}})
		}
		return
	}
	if respBody == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": fmt.Sprintf("empty response from upstream, statusCode=%d", statusCode)}})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

func (h *ProxyHandler) setRateLimitHeaders(c *gin.Context, token *model.Token) {
	// 设置默认Rate Limit头，确保所有请求都有响应头
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
			usage, err := h.rateLimitRepo.GetUsage(token.ID, idx)
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

func (h *ProxyHandler) HandleStream(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Missing authorization header"}})
		return
	}

	var apiKey string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		apiKey = authHeader[7:]
	} else {
		apiKey = authHeader
	}

	token, err := h.tokenSvc.GetByKey(apiKey)
	if err != nil || token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Invalid API key"}})
		return
	}

	if token.Enabled != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "API key is disabled"}})
		return
	}

	if token.AutoDisabled == 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": fmt.Sprintf("API key is auto-disabled: %s", token.AutoDisableReason)}})
		return
	}

	h.setRateLimitHeaders(c, token)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read request body"}})
		return
	}

	h.HandleStreamWithBody(c, token, body)
}

func (h *ProxyHandler) HandleStreamWithBody(c *gin.Context, token *model.Token, body []byte) {
	startTime := time.Now()
	streamResp, statusCode, err := h.dispatcher.DispatchStreamDirect(token, body)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "rate limit") {
			h.setRateLimitHeaders(c, token)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"message": errMsg}})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": errMsg}})
		}
		return
	}
	defer streamResp.Resp.Body.Close()

	h.setRateLimitHeaders(c, token)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	c.Status(statusCode)

	flush := func() {
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}

	var responseBody []byte
	buf := make([]byte, 4096)
	chunkBuf := bytes.NewBuffer(nil)
	for {
		n, err := streamResp.Resp.Body.Read(buf)
		if n > 0 {
			chunkBuf.Write(buf[:n])
			responseBody = append(responseBody, buf[:n]...)
		}
		if chunkBuf.Len() >= 1024 {
			c.Writer.Write(chunkBuf.Bytes())
			chunkBuf.Reset()
			flush()
		}
		if err != nil {
			if chunkBuf.Len() > 0 {
				c.Writer.Write(chunkBuf.Bytes())
				flush()
			}
			if err == io.EOF {
				break
			}
			break
		}
	}

	latency := int(time.Since(startTime).Milliseconds())
	tokenUsed := h.dispatcher.ParseStreamUsage(responseBody)

	go h.dispatcher.LogStreamCompletion(token.ID, token.Name, streamResp.ChannelName, streamResp.ModelName, statusCode, latency, tokenUsed)
}

func (h *ProxyHandler) HandleModels(c *gin.Context) {
	models, err := h.dispatcher.ListEnabledModels()
	if err != nil || models == nil {
		c.JSON(http.StatusOK, gin.H{
			"object": "list",
			"data":   []gin.H{},
		})
		return
	}

	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":       m.Name,
			"object":   "model",
			"created":  m.CreatedAt.Unix(),
			"owned_by": m.ChannelName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func (h *ProxyHandler) HandleChatCompletions(c *gin.Context) {
	h.Handle(c)
}

func (h *ProxyHandler) HandleEmbeddings(c *gin.Context) {
	h.Handle(c)
}

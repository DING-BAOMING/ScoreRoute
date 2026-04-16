package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type ProxyHandler struct {
	dispatcher *service.Dispatcher
	tokenSvc   *service.TokenService
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		dispatcher: service.NewDispatcher(),
		tokenSvc:   service.NewTokenService(),
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
		return
	}
	if respBody == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": fmt.Sprintf("empty response from upstream, statusCode=%d", statusCode)}})
		return
	}

	c.Data(statusCode, "application/json", respBody)
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

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Failed to read request body"}})
		return
	}

	h.HandleStreamWithBody(c, token, body)
}

func (h *ProxyHandler) HandleStreamWithBody(c *gin.Context, token *model.Token, body []byte) {
	c.Header("Content-Type", "text/event-stream")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	statusCode, err := h.dispatcher.DispatchStreamToWriter(c.Writer, token, body)
	if err != nil {
		log.Printf("[ERROR] DispatchStreamToWriter failed: %v", err)
	}
	if statusCode != 200 {
		c.JSON(statusCode, gin.H{"error": gin.H{"message": fmt.Sprintf("upstream error: status %d", statusCode)}})
	}
}

func (h *ProxyHandler) HandleModels(c *gin.Context) {
	models, err := h.dispatcher.ListEnabledModels()
	if err != nil || models == nil {
		c.JSON(http.StatusOK, gin.H{
			"object": "list",
			"data": []gin.H{},
		})
		return
	}

	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":         m.Name,
			"object":     "model",
			"created":    m.CreatedAt.Unix(),
			"owned_by":   m.ChannelName,
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

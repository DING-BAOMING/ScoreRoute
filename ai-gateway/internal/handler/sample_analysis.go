package handler

import (
	"net/http"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

type SampleAnalysisHandler struct {
	svc *service.SampleAnalysisService
}

func NewSampleAnalysisHandler() *SampleAnalysisHandler {
	return &SampleAnalysisHandler{
		svc: service.NewSampleAnalysisService(),
	}
}

func (h *SampleAnalysisHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "not configured", Data: nil})
		return
	}
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: cfg})
}

func (h *SampleAnalysisHandler) SaveConfig(c *gin.Context) {
	var req model.SampleAnalysisConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	if req.Enabled != nil {
		switch v := req.Enabled.(type) {
		case bool:
			if v {
				req.Enabled = 1
			} else {
				req.Enabled = 0
			}
		case float64:
			req.Enabled = int(v)
		case int:
			req.Enabled = v
		}
	}

	if err := h.svc.SaveConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "保存成功"})
}

func (h *SampleAnalysisHandler) TestConfig(c *gin.Context) {
	var req model.SampleAnalysisConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	if req.Enabled != nil {
		switch v := req.Enabled.(type) {
		case bool:
			if v {
				req.Enabled = 1
			} else {
				req.Enabled = 0
			}
		case float64:
			req.Enabled = int(v)
		case int:
			req.Enabled = v
		}
	}

	success, msg, err := h.svc.TestConnection(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	if !success {
		c.JSON(http.StatusOK, model.APIResponse{Code: 1, Message: msg})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: msg})
}

func (h *SampleAnalysisHandler) RunAnalysis(c *gin.Context) {
	analyzed, err := h.svc.RunScheduledAnalysis(20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "分析完成", Data: map[string]interface{}{"analyzed": analyzed}})
}

func (h *SampleAnalysisHandler) GetLogs(c *gin.Context) {
	logs, err := h.svc.GetLogs(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: logs})
}

func (h *SampleAnalysisHandler) GetLogStats(c *gin.Context) {
	stats, err := h.svc.GetLogStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: stats})
}

func (h *SampleAnalysisHandler) GetRatings(c *gin.Context) {
	ratings, err := h.svc.GetRatings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: ratings})
}

func (h *SampleAnalysisHandler) UpdateRating(c *gin.Context) {
	var req model.SampleRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: err.Error()})
		return
	}

	if err := h.svc.UpdateRating(req.ModelKey, req.Score); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "更新成功"})
}

func (h *SampleAnalysisHandler) GetRatingsMap(c *gin.Context) {
	ratings, err := h.svc.GetRatingsMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "success", Data: ratings})
}

func (h *SampleAnalysisHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/sample-analysis/config", h.GetConfig)
	rg.POST("/sample-analysis/config", h.SaveConfig)
	rg.POST("/sample-analysis/config/test", h.TestConfig)
	rg.POST("/sample-analysis/run", h.RunAnalysis)
	rg.GET("/sample-analysis/logs", h.GetLogs)
	rg.GET("/sample-analysis/logs/stats", h.GetLogStats)
	rg.GET("/sample-analysis/ratings", h.GetRatings)
	rg.GET("/sample-analysis/ratings/map", h.GetRatingsMap)
	rg.PUT("/sample-analysis/ratings", h.UpdateRating)
}

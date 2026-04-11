package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/model"
	"ai-gateway/internal/service"
)

type UserRatingHandler struct {
	service *service.UserRatingService
}

func NewUserRatingHandler() *UserRatingHandler {
	return &UserRatingHandler{
		service: service.NewUserRatingService(),
	}
}

func (h *UserRatingHandler) Upsert(c *gin.Context) {
	var req model.UserRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的请求: " + err.Error()})
		return
	}

	if req.UserRating < 1 || req.UserRating > 100 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "评分必须在1-100之间"})
		return
	}

	if err := h.service.UpsertRating(&req); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "保存成功"})
}

func (h *UserRatingHandler) List(c *gin.Context) {
	deduplicated := c.Query("deduplicated")
	if deduplicated == "true" {
		ratings, err := h.service.GetDeduplicatedRatings()
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: ratings})
		return
	}
	
	ratings, err := h.service.GetAllUserRatingsWithDefaults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "查询成功", Data: ratings})
}

func (h *UserRatingHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	if err := h.service.DeleteRating(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: "删除成功"})
}

package handler

import (
	"encoding/json"
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

type UserRatingUpsertRequest struct {
	ModelName  string `json:"model_name"`
	UserRating int    `json:"user_rating"`
	Rating     int    `json:"rating"`
}

func (r *UserRatingUpsertRequest) ParseAndValidate() (*model.UserRatingRequest, error) {
	userRating := r.UserRating
	if userRating == 0 && r.Rating != 0 {
		userRating = r.Rating
	}

	if userRating < 1 || userRating > 100 {
		return nil, &json.UnmarshalTypeError{}
	}

	return &model.UserRatingRequest{
		ModelName:  r.ModelName,
		UserRating: userRating,
	}, nil
}

func (h *UserRatingHandler) Upsert(c *gin.Context) {
	var req UserRatingUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "无效的请求: 请使用 user_rating 字段(值范围1-100)"})
		return
	}

	validReq, err := req.ParseAndValidate()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "评分必须在1-100之间，请使用 user_rating 或 rating 字段"})
		return
	}

	if validReq.ModelName == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Code: 400, Message: "model_name 为必填字段"})
		return
	}

	if err := h.service.UpsertRating(validReq); err != nil {
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

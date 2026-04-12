package router

import (
	"github.com/gin-gonic/gin"

	"ai-gateway/internal/handler"
	"ai-gateway/internal/middleware"
)

func Setup() *gin.Engine {
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.LoadHTMLGlob("./web/dist/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.POST("/api/auth/login", handler.NewAuthHandler().Login)
	r.GET("/health", handler.NewAuthHandler().HealthCheck)
	r.GET("/api/health", handler.NewAuthHandler().HealthCheck)

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		api.GET("/auth/validate", handler.NewAuthHandler().Validate)

		channels := api.Group("/channels")
		{
			channels.GET("", handler.NewChannelHandler().List)
			channels.POST("", handler.NewChannelHandler().Create)
			channels.POST("/test-credentials", handler.NewChannelHandler().TestCredentials)
			channels.GET("/:id", handler.NewChannelHandler().Get)
			channels.PUT("/:id", handler.NewChannelHandler().Update)
			channels.DELETE("/:id", handler.NewChannelHandler().Delete)
			channels.PUT("/:id/enabled", handler.NewChannelHandler().SetEnabled)
			channels.GET("/:id/models", handler.NewChannelHandler().FetchModels)
		}

		models := api.Group("/models")
		{
			models.GET("", handler.NewModelHandler().List)
			models.POST("", handler.NewModelHandler().Create)
			models.POST("/batch", handler.NewModelHandler().BatchCreate)
			models.POST("/test/:id", handler.NewModelHandler().Test)
			models.GET("/:id", handler.NewModelHandler().Get)
			models.PUT("/:id", handler.NewModelHandler().Update)
			models.DELETE("/:id", handler.NewModelHandler().Delete)
			models.PUT("/:id/enabled", handler.NewModelHandler().SetEnabled)
			models.GET("/channel/:channel_id", handler.NewModelHandler().ListByChannel)
		}

		tokens := api.Group("/tokens")
		{
			tokens.GET("", handler.NewTokenHandler().List)
			tokens.POST("", handler.NewTokenHandler().Create)
			tokens.GET("/:id", handler.NewTokenHandler().Get)
			tokens.PUT("/:id", handler.NewTokenHandler().Update)
			tokens.DELETE("/:id", handler.NewTokenHandler().Delete)
			tokens.PUT("/:id/enabled", handler.NewTokenHandler().SetEnabled)
			tokens.POST("/:id/regenerate", handler.NewTokenHandler().RegenerateKey)
		}

		logs := api.Group("/logs")
		{
			logs.GET("", handler.NewLogHandler().List)
			logs.GET("/stats", handler.NewLogHandler().Stats)
			logs.GET("/dashboard", handler.NewLogHandler().Dashboard)
			logs.DELETE("/cleanup", handler.NewLogHandler().Cleanup)
			logs.GET("/model-stats", handler.NewLogHandler().ModelStats)
		}

		userRatings := api.Group("/user-ratings")
		{
			userRatings.GET("", handler.NewUserRatingHandler().List)
			userRatings.POST("", handler.NewUserRatingHandler().Upsert)
			userRatings.DELETE("/:id", handler.NewUserRatingHandler().Delete)
		}

		samples := api.Group("/samples")
		{
			samples.GET("", handler.NewSampleHandler().List)
			samples.GET("/stats", handler.NewSampleHandler().Stats)
			samples.GET("/:modelKey", handler.NewSampleHandler().Get)
			samples.DELETE("/:id", handler.NewSampleHandler().Delete)
			samples.DELETE("/cleanup", handler.NewSampleHandler().Cleanup)
		}

		sampleAnalysis := api.Group("/sample-analysis")
		{
			sampleAnalysisHandler := handler.NewSampleAnalysisHandler()
			sampleAnalysis.GET("/config", sampleAnalysisHandler.GetConfig)
			sampleAnalysis.POST("/config", sampleAnalysisHandler.SaveConfig)
			sampleAnalysis.POST("/config/test", sampleAnalysisHandler.TestConfig)
			sampleAnalysis.POST("/run", sampleAnalysisHandler.RunAnalysis)
			sampleAnalysis.GET("/logs", sampleAnalysisHandler.GetLogs)
			sampleAnalysis.GET("/logs/stats", sampleAnalysisHandler.GetLogStats)
			sampleAnalysis.GET("/ratings", sampleAnalysisHandler.GetRatings)
			sampleAnalysis.GET("/ratings/map", sampleAnalysisHandler.GetRatingsMap)
			sampleAnalysis.PUT("/ratings", sampleAnalysisHandler.UpdateRating)
		}

		systemConfig := api.Group("/system-config")
		{
			systemConfigHandler := handler.NewSystemConfigHandler()
			systemConfig.GET("", systemConfigHandler.Get)
			systemConfig.PUT("", systemConfigHandler.Update)
		}

		extraRating := api.Group("/extra-rating")
		{
			extraRatingHandler := handler.NewExtraRatingHandler()
			extraRating.GET("/config", extraRatingHandler.GetConfig)
			extraRating.PUT("/config", extraRatingHandler.SetConfig)
			extraRating.GET("/records", extraRatingHandler.GetRecords)
			extraRating.GET("/model-scores", extraRatingHandler.GetAllModelExtraScores)
			extraRating.DELETE("/records", extraRatingHandler.ClearRecords)
			extraRating.DELETE("/records/:id", extraRatingHandler.DeleteRecord)
		}

		modelRating := api.Group("/model-rating")
		{
			modelRatingHandler := handler.NewModelRatingHandler()
			modelRating.GET("/weights", modelRatingHandler.GetWeights)
			modelRating.PUT("/weights", modelRatingHandler.UpdateWeights)
			modelRating.GET("/cost-time", modelRatingHandler.GetCostTimeRatings)
		}
	}

	v1 := r.Group("/v1")
	{
		v1.POST("/chat/completions", func(c *gin.Context) {
			proxyHandler := handler.NewProxyHandler()
			proxyHandler.Handle(c)
		})
		v1.GET("/models", func(c *gin.Context) {
			proxyHandler := handler.NewProxyHandler()
			proxyHandler.HandleModels(c)
		})
	}

	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	return r
}

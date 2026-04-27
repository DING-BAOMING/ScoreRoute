package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ai-gateway/internal/handler"
	"ai-gateway/internal/middleware"
)

func Setup() *gin.Engine {
	r := gin.Default()

	r.Use(middleware.CORSMiddleware())

	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.StaticFile("/logo.png", "./web/dist/logo.png")
	r.StaticFile("/docs/README.md", "./docs/README.md")
	r.StaticFile("/docs/getting-started.md", "./docs/getting-started.md")
	r.StaticFile("/docs/guide.md", "./docs/guide.md")
	r.StaticFile("/docs/api.md", "./docs/api.md")
	r.StaticFile("/docs/faq.md", "./docs/faq.md")
	r.StaticFile("/docs/changelog.md", "./docs/changelog.md")
	r.StaticFile("/docs/dev/setup.md", "./docs/dev/setup.md")
	r.StaticFile("/docs/dev/architecture.md", "./docs/dev/architecture.md")
	r.StaticFile("/docs/dev/plan.md", "./docs/dev/plan.md")
	r.StaticFile("/docs/dev/plan_full.md", "./docs/dev/plan_full.md")
	r.StaticFile("/docs/dev/workflow.md", "./docs/dev/workflow.md")
	r.LoadHTMLGlob("./web/dist/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/markdown")
		c.File("./docs/README.md")
	})

	r.POST("/api/auth/login", handler.NewAuthHandler().Login)
	r.POST("/api/auth/passwordless-login", handler.NewAuthHandler().PasswordLessLogin)
	r.GET("/api/auth/setup-status", handler.NewAuthHandler().CheckSetupStatus)
	r.POST("/api/system-config/setup-password", handler.NewSystemConfigHandler().SetupPassword)
	r.PUT("/api/system-config/password-less", handler.NewSystemConfigHandler().EnablePasswordLessMode)
	r.GET("/health", handler.NewAuthHandler().HealthCheck)
	r.GET("/api/health", handler.NewAuthHandler().HealthCheck)

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	api.Use(middleware.RateLimitHeadersMiddleware())
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
			channels.PUT("/:id/rate-limit", handler.NewChannelHandler().SetRateLimit)
			channels.GET("/:id/models", handler.NewChannelHandler().FetchModels)
			channels.GET("/rate-limit", handler.NewChannelHandler().List)
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
			models.PUT("/:id/rate-limit", handler.NewModelHandler().SetRateLimit)
			models.GET("/channel/:channel_id", handler.NewModelHandler().ListByChannel)
			models.GET("/rate-limit", handler.NewModelHandler().List)
		}

		tokens := api.Group("/tokens")
		{
			tokens.GET("", handler.NewTokenHandler().List)
			tokens.POST("", handler.NewTokenHandler().Create)
			tokens.POST("/batch", handler.NewTokenHandler().BatchCreate)
			tokens.GET("/:id", handler.NewTokenHandler().Get)
			tokens.PUT("/:id", handler.NewTokenHandler().Update)
			tokens.DELETE("/:id", handler.NewTokenHandler().Delete)
			tokens.PUT("/:id/enabled", handler.NewTokenHandler().SetEnabled)
			tokens.PUT("/:id/rate-limit", handler.NewTokenHandler().SetRateLimit)
			tokens.POST("/:id/regenerate", handler.NewTokenHandler().RegenerateKey)
			tokens.GET("/rate-limit", handler.NewTokenHandler().List)
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
			systemConfig.PUT("/dispatch-mode", systemConfigHandler.UpdateDispatchMode)
			systemConfig.GET("/setup-status", systemConfigHandler.CheckSetupStatus)
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
			modelRating.GET("/scores", modelRatingHandler.GetAllScores)
		}

		extraRatings := api.Group("/extra-ratings")
		{
			extraRatingHandler := handler.NewExtraRatingHandler()
			extraRatings.GET("", extraRatingHandler.GetRecords)
			extraRatings.GET("/config", extraRatingHandler.GetConfig)
			extraRatings.PUT("/config", extraRatingHandler.SetConfig)
			extraRatings.PUT("/penalty", extraRatingHandler.UpdatePenalty)
			extraRatings.PUT("/reward", extraRatingHandler.UpdateReward)
			extraRatings.GET("/records", extraRatingHandler.GetRecords)
			extraRatings.DELETE("/records", extraRatingHandler.ClearRecords)
			extraRatings.DELETE("/records/:id", extraRatingHandler.DeleteRecord)
			extraRatings.GET("/model-scores", extraRatingHandler.GetAllModelExtraScores)
		}

		modelRatings := api.Group("/model-ratings")
		{
			modelRatingHandler := handler.NewModelRatingHandler()
			modelRatings.GET("", modelRatingHandler.GetAllScores)
			modelRatings.GET("/weights", modelRatingHandler.GetWeights)
			modelRatings.PUT("/weights", modelRatingHandler.UpdateWeights)
			modelRatings.GET("/cost-time", modelRatingHandler.GetCostTimeRatings)
			modelRatings.GET("/scores", modelRatingHandler.GetAllScores)
		}

		sampleAnalysisConfig := api.Group("/sample-analysis-config")
		{
			sampleAnalysisHandler := handler.NewSampleAnalysisHandler()
			sampleAnalysisConfig.GET("", sampleAnalysisHandler.GetConfig)
			sampleAnalysisConfig.PUT("", sampleAnalysisHandler.SaveConfig)
		}

		invocations := api.Group("/invocations")
		{
			invocations.GET("", handler.NewLogHandler().List)
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
		v1.POST("/models", func(c *gin.Context) {
			proxyHandler := handler.NewProxyHandler()
			proxyHandler.HandleModels(c)
		})
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v2/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API路由不存在"})
			return
		}
		c.File("./web/dist/index.html")
	})

	return r
}

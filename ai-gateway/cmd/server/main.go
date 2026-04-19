package main

import (
	"log"
	"os"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/repository"
	"ai-gateway/internal/router"
	"ai-gateway/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := os.MkdirAll(cfg.LogPath, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	if err := repository.InitDB(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized successfully")

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("Running scheduled log cleanup (24 hours)")
			if err := service.NewLogService().Cleanup(1); err != nil {
				log.Printf("Log cleanup failed: %v", err)
			}
			log.Println("Running scheduled sample cleanup (7 days)")
			if _, err := service.NewSampleService().CleanupOldSamples(7); err != nil {
				log.Printf("Sample cleanup failed: %v", err)
			}
			log.Println("Running scheduled sample analysis cleanup (7 days)")
			if err := service.NewSampleAnalysisService().CleanupExpiredRatings(); err != nil {
				log.Printf("Sample analysis rating cleanup failed: %v", err)
			}
			log.Println("Running scheduled extra rating cleanup")
			if err := service.NewExtraRatingService().CleanupExpired(); err != nil {
				log.Printf("Extra rating cleanup failed: %v", err)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(2 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("Running scheduled sample analysis")
			analyzed, err := service.NewSampleAnalysisService().RunScheduledAnalysis(20)
			if err != nil {
				log.Printf("Sample analysis failed: %v", err)
			} else {
				log.Printf("Sample analysis completed: %d samples analyzed and deleted", analyzed)
			}
		}
	}()

	dispatcher := service.NewDispatcher()
	dispatcher.StartAutoEnableScheduler()

	r := router.Setup()

	port := cfg.ServerPort
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting ScoreRoute on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"port-scanner/internal/application"
	"port-scanner/internal/domain"
	"port-scanner/internal/infrastructure/banner"
	"port-scanner/internal/infrastructure/config"
	"port-scanner/internal/infrastructure/database"
	httphandler "port-scanner/internal/infrastructure/http"
	"port-scanner/internal/infrastructure/queue"
	"port-scanner/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	log.InitLogger("port-scanner")
	defer log.L().Sync()
	log.L().Info("Starting port-scanner service")

	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.L().Fatal("Failed to load configuration", zap.Error(err))
	}

	// Create domain services
	scanConfig := cfg.ToDomainScanConfig()
	scanner := domain.NewScannerService(scanConfig)

	// Create and configure optimized banner grabber with worker pool
	bannerGrabber := banner.NewBannerGrabber(
		scanConfig.ZGrabConcurrency,
		scanConfig.BannerTimeout,
		scanConfig.PriorityPorts,
	)
	scanner.SetBannerGrabber(bannerGrabber)

	// Create and configure ZGrab2 banner service as fallback
	bannerService := banner.NewZGrabBannerService(scanConfig.BannerTimeout)
	scanner.SetBannerGrabber(bannerService)

	// Create MongoDB manager if enabled
	var dbManager *database.MongoDBManager
	if cfg.MongoDB.EnableDatabase {
		dbManager, err = database.NewMongoDBManager(
			cfg.MongoDB.ConnectionString,
			cfg.MongoDB.DatabaseName,
			cfg.MongoDB.CollectionName,
		)
		if err != nil {
			log.L().Error("Failed to connect to MongoDB", zap.Error(err))
			log.L().Warn("Continuing without MongoDB - results will not be persisted")
		} else {
			log.L().Info("MongoDB connected successfully",
				zap.String("database", cfg.MongoDB.DatabaseName),
				zap.String("collection", cfg.MongoDB.CollectionName))
			defer dbManager.Close()
		}
	}

	// Create queue manager
	queueManager, err := queue.NewRabbitMQManager(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.IPQueue,
		cfg.RabbitMQ.ScanResultQueue,
		cfg.RabbitMQ.EnrichmentQueue,
		cfg.RabbitMQ.ServiceAnalysisQueue,
	)
	if err != nil {
		log.L().Fatal("Failed to create queue manager", zap.Error(err))
	}
	defer queueManager.Close()

	// Configure queue manager with scan handler, config, and MongoDB
	queueManager.SetScanHandler(scanner.ScanIP)
	queueManager.SetScanConfig(scanConfig)
	if dbManager != nil {
		queueManager.SetMongoDBManager(dbManager)
	}

	// Create application services
	scanEngine := application.NewScanEngineService(scanner, queueManager, scanConfig)

	// Start the scanning engine
	if err := scanEngine.StartScanning(); err != nil {
		log.L().Fatal("Failed to start scanning engine", zap.Error(err))
	}

	// Create HTTP server
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Create and register HTTP handlers
	httpHandler := httphandler.NewHandler(scanEngine, scanner, dbManager)
	httpHandler.RegisterRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.L().Info("HTTP server started", zap.String("host", cfg.Server.Host), zap.String("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.L().Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.L().Info("Shutting down server...")

	// Stop the scanning engine
	scanEngine.StopScanning()

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.L().Info("Server exited")
}

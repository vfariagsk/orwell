package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ip-generator/internal/application"
	"ip-generator/internal/domain"
	"ip-generator/internal/infrastructure/config"
	"ip-generator/internal/infrastructure/http"
	"ip-generator/internal/infrastructure/queue"
	"ip-generator/pkg/log"

	"go.uber.org/zap"
)

func main() {
	log.InitLogger("ip-generator")
	defer log.L().Sync()
	log.L().Info("Starting ip-generator service")

	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.L().Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize domain services
	ipGenerator := domain.NewIPGeneratorService()

	// Initialize infrastructure
	queuePublisher, err := queue.NewRabbitMQPublisher(cfg.RabbitMQ.URL, cfg.RabbitMQ.Queue)
	if err != nil {
		log.L().Fatal("Failed to create RabbitMQ publisher", zap.Error(err))
	}
	defer queuePublisher.Close()

	// Initialize application service
	appService := application.NewIPGenerationService(ipGenerator, queuePublisher)

	// Initialize HTTP server
	server := http.NewServer(cfg.Server.Port, appService)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.L().Error("Server error", zap.Error(err))
		}
	}()

	log.L().Info("HTTP server started", zap.String("host", cfg.Server.Host), zap.String("port", cfg.Server.Port))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.L().Info("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Stop(ctx); err != nil {
		log.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.L().Info("Server exited")
}

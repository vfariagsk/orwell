package http

import (
	"context"
	"net/http"
	"time"

	"ip-generator/internal/application"
	"ip-generator/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	server  *http.Server
	handler *Handler
	router  *gin.Engine
}

// NewServer creates a new HTTP server
func NewServer(port string, service *application.IPGenerationService) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestIDMiddleware())
	router.Use(rateLimitMiddleware())

	// Create handler
	handler := NewHandler(service)

	// Setup routes
	handler.SetupRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		server:  server,
		handler: handler,
		router:  router,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.L().Info("Starting HTTP server", zap.String("event", "server_start"), zap.String("port", s.server.Addr))
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.L().Info("Stopping HTTP server", zap.String("event", "server_stop"))
	return s.server.Shutdown(ctx)
}

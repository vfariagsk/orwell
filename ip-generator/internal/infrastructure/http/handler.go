package http

import (
	"net/http"
	"strconv"

	"ip-generator/internal/application"
	"ip-generator/pkg/log"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the IP generator service
type Handler struct {
	service *application.IPGenerationService
}

// NewHandler creates a new HTTP handler
func NewHandler(service *application.IPGenerationService) *Handler {
	return &Handler{
		service: service,
	}
}

// GenerateIPsRequest represents the request body for generating IPs
type GenerateIPsRequest struct {
	Count     int `json:"count" binding:"required,min=1"`
	BatchSize int `json:"batch_size" binding:"min=1"`
}

// GenerateSequentialIPsRequest represents the request body for generating sequential IPs
type GenerateSequentialIPsRequest struct {
	StartIP   string `json:"start_ip" binding:"required"`
	Count     int    `json:"count" binding:"required,min=1"`
	BatchSize int    `json:"batch_size" binding:"min=1"`
}

// Response represents a generic API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GenerateRandomIPs handles requests to generate random IP addresses
func (h *Handler) GenerateRandomIPs(c *gin.Context) {
	var req GenerateIPsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.L().Warn("Invalid IP generation request", zap.String("event", "generateip_invalid_request"), zap.Error(err))
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	log.L().Info("Received IP generation request", zap.String("event", "generateip_request"), zap.Any("params", req))

	// Set default batch size if not provided
	if req.BatchSize <= 0 {
		req.BatchSize = 100
	}

	err := h.service.GenerateAndPublishIPs(req.Count, req.BatchSize)
	if err != nil {
		log.L().Error("IP generation failed", zap.String("event", "generateip_failed"), zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate and publish IPs: " + err.Error(),
		})
		return
	}

	log.L().Info("IPs generated and published successfully", zap.String("event", "generateip_success"), zap.Int("count", req.Count), zap.Int("batch_size", req.BatchSize))

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "IPs generated and published successfully",
		Data: gin.H{
			"count":      req.Count,
			"batch_size": req.BatchSize,
		},
	})
}

// GenerateSequentialIPs handles requests to generate sequential IP addresses
func (h *Handler) GenerateSequentialIPs(c *gin.Context) {
	var req GenerateSequentialIPsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.L().Warn("Invalid IP generation request", zap.String("event", "generatesequentialip_invalid_request"), zap.Error(err))
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	log.L().Info("Received IP generation request", zap.String("event", "generatesequentialip_request"), zap.Any("params", req))

	// Set default batch size if not provided
	if req.BatchSize <= 0 {
		req.BatchSize = 100
	}

	err := h.service.GenerateAndPublishSequentialIPs(req.StartIP, req.Count, req.BatchSize)
	if err != nil {
		log.L().Error("IP generation failed", zap.String("event", "generatesequentialip_failed"), zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate and publish sequential IPs: " + err.Error(),
		})
		return
	}

	log.L().Info("Sequential IPs generated and published successfully", zap.String("event", "generatesequentialip_success"), zap.String("start_ip", req.StartIP), zap.Int("count", req.Count), zap.Int("batch_size", req.BatchSize))

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Sequential IPs generated and published successfully",
		Data: gin.H{
			"start_ip":   req.StartIP,
			"count":      req.Count,
			"batch_size": req.BatchSize,
		},
	})
}

// GenerateIPsWithQueryParams handles requests with query parameters
func (h *Handler) GenerateIPsWithQueryParams(c *gin.Context) {
	countStr := c.Query("count")
	batchSizeStr := c.Query("batch_size")

	if countStr == "" {
		log.L().Warn("Invalid IP generation request", zap.String("event", "generatequeryip_invalid_count"), zap.Error(nil))
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "count parameter is required",
		})
		return
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		log.L().Warn("Invalid IP generation request", zap.String("event", "generatequeryip_invalid_count"), zap.Error(err))
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "count must be a positive integer",
		})
		return
	}

	batchSize := 100 // default
	if batchSizeStr != "" {
		batchSize, err = strconv.Atoi(batchSizeStr)
		if err != nil || batchSize <= 0 {
			log.L().Warn("Invalid IP generation request", zap.String("event", "generatequeryip_invalid_batchsize"), zap.Error(err))
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "batch_size must be a positive integer",
			})
			return
		}
	}

	err = h.service.GenerateAndPublishIPs(count, batchSize)
	if err != nil {
		log.L().Error("IP generation failed", zap.String("event", "generatequeryip_failed"), zap.Error(err))
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate and publish IPs: " + err.Error(),
		})
		return
	}

	log.L().Info("IPs generated and published successfully", zap.String("event", "generatequeryip_success"), zap.Int("count", count), zap.Int("batch_size", batchSize))

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "IPs generated and published successfully",
		Data: gin.H{
			"count":      count,
			"batch_size": batchSize,
		},
	})
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Service is healthy",
		Data: gin.H{
			"status": "ok",
		},
	})
}

// GetServiceInfo returns information about the service
func (h *Handler) GetServiceInfo(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "IP Generator Service Information",
		Data: gin.H{
			"service": "IP Generator Microservice",
			"version": "1.0.0",
			"endpoints": gin.H{
				"health":              "/health",
				"info":                "/api/v1/info",
				"generate_random":     "/api/v1/ips/generate",
				"generate_sequential": "/api/v1/ips/generate/sequential",
				"generate_query":      "/api/v1/ips/generate/query",
			},
		},
	})
}

// SetupRoutes configures the Gin router with all the routes
func (h *Handler) SetupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", h.HealthCheck)

	// Service info
	router.GET("/api/v1/info", h.GetServiceInfo)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		ips := v1.Group("/ips")
		{
			ips.POST("/generate", h.GenerateRandomIPs)
			ips.POST("/generate/sequential", h.GenerateSequentialIPs)
			ips.GET("/generate/query", h.GenerateIPsWithQueryParams)
		}
	}
}

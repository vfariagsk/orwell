package http

import (
	"net/http"
	"sync"
	"time"

	"port-scanner/internal/application"
	"port-scanner/internal/domain"
	"port-scanner/internal/infrastructure/database"
	"port-scanner/pkg/log"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the port scanner
type Handler struct {
	scanEngine *application.ScanEngineService
	scanner    domain.Scanner
	dbManager  *database.MongoDBManager
}

// NewHandler creates a new HTTP handler
func NewHandler(scanEngine *application.ScanEngineService, scanner domain.Scanner, dbManager *database.MongoDBManager) *Handler {
	return &Handler{
		scanEngine: scanEngine,
		scanner:    scanner,
		dbManager:  dbManager,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.GET("/health", h.HealthCheck)
		api.GET("/stats", h.GetStats)
		api.GET("/banner-stats", h.GetBannerStats)
		api.GET("/status/:ip", h.GetScanStatus)
		api.POST("/scan", h.ScanIP)
		api.POST("/scan/batch", h.ScanBatch)
		api.GET("/ports/:ip", h.GetOpenPorts)

		// MongoDB endpoints
		api.GET("/db/stats", h.GetDatabaseStats)
		api.GET("/db/result/:ip", h.GetDatabaseResult)
		api.GET("/db/batch/:batch_id", h.GetDatabaseBatchResults)
		api.GET("/db/search", h.SearchDatabaseResults)
	}
}

// HealthCheck returns the health status of the service
func (h *Handler) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "port-scanner",
	}

	// Add MongoDB status if available
	if h.dbManager != nil {
		health["mongodb"] = "connected"
	} else {
		health["mongodb"] = "disabled"
	}

	c.JSON(http.StatusOK, health)
}

// GetStats returns scanning statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats := h.scanEngine.GetScanStats()

	response := gin.H{
		"total_scanned":     stats.TotalScanned,
		"successful_scans":  stats.SuccessfulScans,
		"failed_scans":      stats.FailedScans,
		"average_scan_time": stats.AverageScanTime.String(),
		"start_time":        stats.StartTime.Unix(),
		"last_scan_time":    stats.LastScanTime.Unix(),
		"uptime":            time.Since(stats.StartTime).String(),
	}

	// Add database stats if available
	if h.dbManager != nil {
		if dbStats, err := h.dbManager.GetScanStats(); err == nil {
			response["database_stats"] = dbStats
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetBannerStats returns banner grabbing statistics
func (h *Handler) GetBannerStats(c *gin.Context) {
	// Try to get banner stats from scanner service
	if scannerService, ok := h.scanner.(*domain.ScannerService); ok {
		stats := scannerService.GetBannerStats()
		c.JSON(http.StatusOK, stats)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Banner statistics not available",
	})
}

// GetScanStatus returns the scan status for a specific IP
func (h *Handler) GetScanStatus(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	result, err := h.scanEngine.GetScanStatus(ip)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ip":            result.IP,
		"status":        result.Status,
		"is_up":         result.IsUp,
		"ping_time":     result.PingTime.String(),
		"scan_start":    result.ScanStartTime.Unix(),
		"scan_end":      result.ScanEndTime.Unix(),
		"scan_duration": result.GetScanDuration().String(),
		"total_ports":   len(result.Ports),
		"open_ports":    len(result.GetOpenPorts()),
		"batch_id":      result.BatchID,
		"error":         result.Error,
	})
}

// ScanIPRequest represents a single IP scan request
type ScanIPRequest struct {
	IP      string `json:"ip" binding:"required"`
	Ports   []int  `json:"ports,omitempty"`
	BatchID string `json:"batch_id,omitempty"`
}

// ScanIP scans a single IP address
func (h *Handler) ScanIP(c *gin.Context) {
	var req ScanIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.L().Warn("Invalid scan request", zap.String("event", "scanip_invalid_request"), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.L().Info("Received scan request", zap.String("event", "scanip_request"), zap.String("ip", req.IP), zap.Any("ports", req.Ports))

	// Create custom config if ports are specified
	config := domain.NewDefaultScanConfig()
	if len(req.Ports) > 0 {
		config.PortRange = req.Ports
	}

	// Perform the scan
	result, err := h.scanner.ScanIP(req.IP, config)
	if err != nil {
		log.L().Error("Scan failed", zap.String("event", "scanip_failed"), zap.String("ip", req.IP), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.L().Info("Scan completed", zap.String("event", "scanip_completed"), zap.String("ip", req.IP), zap.Int("open_ports", len(result.GetOpenPorts())))

	// Update statistics
	h.scanEngine.GetScanStats().UpdateStats(result)

	c.JSON(http.StatusOK, gin.H{
		"ip":            result.IP,
		"status":        result.Status,
		"is_up":         result.IsUp,
		"ping_time":     result.PingTime.String(),
		"scan_duration": result.GetScanDuration().String(),
		"total_ports":   len(result.Ports),
		"open_ports":    len(result.GetOpenPorts()),
		"ports":         h.formatPortsForResponse(result.Ports),
		"batch_id":      result.BatchID,
	})
}

// ScanBatchRequest represents a batch scan request
type ScanBatchRequest struct {
	IPs     []string `json:"ips" binding:"required"`
	Ports   []int    `json:"ports,omitempty"`
	BatchID string   `json:"batch_id,omitempty"`
}

// ScanBatch scans multiple IP addresses
func (h *Handler) ScanBatch(c *gin.Context) {
	var req ScanBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create custom config if ports are specified
	config := domain.NewDefaultScanConfig()
	if len(req.Ports) > 0 {
		config.PortRange = req.Ports
	}

	var results []gin.H
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Limit concurrency
	semaphore := make(chan struct{}, config.Concurrency)

	for _, ip := range req.IPs {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Perform the scan
			result, err := h.scanner.ScanIP(ipAddr, config)

			mu.Lock()
			if err != nil {
				results = append(results, gin.H{
					"ip":     ipAddr,
					"status": "failed",
					"error":  err.Error(),
				})
			} else {
				results = append(results, gin.H{
					"ip":            result.IP,
					"status":        result.Status,
					"is_up":         result.IsUp,
					"ping_time":     result.PingTime.String(),
					"scan_duration": result.GetScanDuration().String(),
					"total_ports":   len(result.Ports),
					"open_ports":    len(result.GetOpenPorts()),
				})
			}
			mu.Unlock()
		}(ip)
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"batch_id":  req.BatchID,
		"total_ips": len(req.IPs),
		"results":   results,
	})
}

// GetOpenPorts returns open ports for a specific IP
func (h *Handler) GetOpenPorts(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	result, err := h.scanEngine.GetScanStatus(ip)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	openPorts := result.GetOpenPorts()
	var ports []gin.H

	for _, port := range openPorts {
		portInfo := gin.H{
			"number":        port.Number,
			"service":       port.Service,
			"banner":        port.Banner,
			"version":       port.Version,
			"response_time": port.ResponseTime.String(),
			"scan_time":     port.ScanTime.Unix(),
		}

		// Add confidence information if available
		if port.BannerInfo != nil {
			portInfo["confidence"] = port.BannerInfo.Confidence
		}

		ports = append(ports, portInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"ip":         ip,
		"open_ports": ports,
		"count":      len(openPorts),
	})
}

// GetDatabaseStats returns statistics from MongoDB
func (h *Handler) GetDatabaseStats(c *gin.Context) {
	if h.dbManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MongoDB not available"})
		return
	}

	stats, err := h.dbManager.GetScanStats()
	if err != nil {
		log.L().Error("Failed to get database stats", zap.String("event", "db_stats_failed"), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve database statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetDatabaseResult returns a scan result from MongoDB
func (h *Handler) GetDatabaseResult(c *gin.Context) {
	if h.dbManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MongoDB not available"})
		return
	}

	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	result, err := h.dbManager.GetScanResult(ip)
	if err != nil {
		log.L().Error("Failed to get database result", zap.String("event", "db_result_failed"), zap.String("ip", ip), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDatabaseBatchResults returns all scan results for a batch from MongoDB
func (h *Handler) GetDatabaseBatchResults(c *gin.Context) {
	if h.dbManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MongoDB not available"})
		return
	}

	batchID := c.Param("batch_id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Batch ID is required"})
		return
	}

	results, err := h.dbManager.GetScanResultsByBatch(batchID)
	if err != nil {
		log.L().Error("Failed to get batch results", zap.String("event", "db_batch_failed"), zap.String("batch_id", batchID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"batch_id": batchID,
		"count":    len(results),
		"results":  results,
	})
}

// SearchDatabaseResults searches scan results in MongoDB
func (h *Handler) SearchDatabaseResults(c *gin.Context) {
	if h.dbManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MongoDB not available"})
		return
	}

	// Get query parameters
	status := c.Query("status")
	isUp := c.Query("is_up")
	limit := c.DefaultQuery("limit", "100")

	// TODO: Implement search functionality in MongoDB manager
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message": "Search functionality not yet implemented",
		"filters": gin.H{
			"status": status,
			"is_up":  isUp,
			"limit":  limit,
		},
	})
}

func (h *Handler) formatPortsForResponse(ports []*domain.Port) []gin.H {
	var formattedPorts []gin.H
	for _, port := range ports {
		portInfo := gin.H{
			"number":        port.Number,
			"service":       port.Service,
			"banner":        port.Banner,
			"version":       port.Version,
			"response_time": port.ResponseTime.String(),
			"scan_time":     port.ScanTime.Unix(),
		}

		// Add confidence information if available
		if port.BannerInfo != nil {
			portInfo["confidence"] = port.BannerInfo.Confidence
		}

		formattedPorts = append(formattedPorts, portInfo)
	}
	return formattedPorts
}

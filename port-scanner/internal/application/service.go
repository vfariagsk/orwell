package application

import (
	"context"
	"fmt"
	"sync"

	"port-scanner/internal/domain"
	"port-scanner/pkg/log"

	"go.uber.org/zap"
)

// ScanEngineService implements the main scanning engine with queue consumption
type ScanEngineService struct {
	scanner      domain.Scanner
	queueManager domain.QueueManager
	config       *domain.ScanConfig
	stats        *domain.ScanStats
	workerPool   chan struct{}
	results      map[string]*domain.ScanResult
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	isRunning    bool
}

// NewScanEngineService creates a new scan engine service
func NewScanEngineService(scanner domain.Scanner, queueManager domain.QueueManager, config *domain.ScanConfig) *ScanEngineService {
	ctx, cancel := context.WithCancel(context.Background())

	return &ScanEngineService{
		scanner:      scanner,
		queueManager: queueManager,
		config:       config,
		stats:        domain.NewScanStats(),
		workerPool:   make(chan struct{}, config.Concurrency),
		results:      make(map[string]*domain.ScanResult),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// StartScanning starts the scanning engine
func (s *ScanEngineService) StartScanning() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scanning engine is already running")
	}

	log.L().Info("Starting port scanner engine", zap.String("event", "engine_start"))

	// Start consuming messages
	err := s.queueManager.ConsumeIPs(s.processMessage)
	if err != nil {
		log.L().Error("Failed to start consuming messages", zap.String("event", "engine_start_failed"), zap.Error(err))
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	s.isRunning = true
	log.L().Info("Port scanner engine started successfully", zap.String("event", "engine_started"))
	return nil
}

// StopScanning stops the scanning engine
func (s *ScanEngineService) StopScanning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	log.L().Info("Stopping port scanner engine", zap.String("event", "engine_stop"))

	// Stop consuming messages
	s.queueManager.Close()

	s.isRunning = false
	log.L().Info("Port scanner engine stopped", zap.String("event", "engine_stopped"))
}

// processMessage handles incoming IP messages from the queue
func (s *ScanEngineService) processMessage(message *domain.QueueMessage) error {
	log.L().Info("Received IP batch", zap.String("event", "batch_received"), zap.String("batch_id", message.BatchID), zap.Int("ip_count", len(message.IPs)))

	// Process each IP in the batch
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.Concurrency)

	for _, ip := range message.IPs {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Scan the IP
			result, err := s.scanner.ScanIP(ipAddr, s.config)
			if err != nil {
				log.L().Error("Scan failed", zap.String("event", "scan_failed"), zap.String("ip", ipAddr), zap.Error(err))
				return
			}

			// Update statistics
			s.stats.UpdateStats(result)

			// Publish scan result
			err = s.queueManager.PublishScanResult(result)
			if err != nil {
				log.L().Error("Failed to publish scan result", zap.String("event", "publish_scan_result_failed"), zap.String("ip", result.IP), zap.Error(err))
			}

			// Publish enrichment message if host is up
			if result.IsUp {
				err = s.queueManager.PublishEnrichmentMessage(result.IP, true, message.BatchID)
				if err != nil {
					log.L().Error("Failed to publish enrichment message", zap.String("event", "publish_enrichment_failed"), zap.String("ip", result.IP), zap.Error(err))
				}
			}

			// Publish service analysis if open ports found
			openPorts := result.GetOpenPorts()
			if len(openPorts) > 0 {
				err = s.queueManager.PublishServiceAnalysis(result.IP, openPorts, message.BatchID)
				if err != nil {
					log.L().Error("Failed to publish service analysis", zap.String("event", "publish_service_analysis_failed"), zap.String("ip", result.IP), zap.Error(err))
				}
			}
		}(ip)
	}

	wg.Wait()

	log.L().Info("Completed processing batch", zap.String("event", "batch_completed"), zap.String("batch_id", message.BatchID))
	return nil
}

// GetScanStatus returns the scan status for a specific IP
func (s *ScanEngineService) GetScanStatus(ip string) (*domain.ScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.results[ip]
	if !exists {
		return nil, fmt.Errorf("no scan result found for IP: %s", ip)
	}

	return result, nil
}

// GetScanStats returns the current scan statistics
func (s *ScanEngineService) GetScanStats() *domain.ScanStats {
	return s.stats
}

// ProcessIP manually processes a single IP (for testing/debugging)
func (s *ScanEngineService) ProcessIP(ip string, batchID string) error {
	// Create a mock message for single IP processing
	message := &domain.QueueMessage{
		IPs:     []string{ip},
		BatchID: batchID,
		Count:   1,
	}
	return s.processMessage(message)
}

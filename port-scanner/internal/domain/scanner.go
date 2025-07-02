package domain

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"port-scanner/internal/infrastructure/ping"
	"port-scanner/pkg/log"

	"go.uber.org/zap"
)

// ScannerService implements the Scanner interface with high concurrency
type ScannerService struct {
	config           *ScanConfig
	stats            *ScanStats
	mu               sync.RWMutex
	bannerGrabber    BannerGrabber
	pingService      *ping.SafePingService
	optimizedGrabber OptimizedBannerGrabber
}

// NewScannerService creates a new scanner service
func NewScannerService(config *ScanConfig) *ScannerService {
	return &ScannerService{
		config:      config,
		stats:       NewScanStats(),
		pingService: ping.NewSafePingService(config.PingTimeout),
	}
}

// SetBannerGrabber sets the banner grabber implementation
func (s *ScannerService) SetBannerGrabber(bg BannerGrabber) {
	s.bannerGrabber = bg
}

// SetOptimizedBannerGrabber sets the optimized banner grabber implementation
func (s *ScannerService) SetOptimizedBannerGrabber(bg OptimizedBannerGrabber) {
	s.optimizedGrabber = bg
}

// PingHost performs a ping to check if the host is up using safe ping service
func (s *ScannerService) PingHost(ip string) (bool, time.Duration, error) {
	result, err := s.pingService.PingHost(ip)
	if err != nil {
		return false, 0, err
	}

	if result.Error != nil {
		return false, result.Duration, result.Error
	}

	return result.IsUp, result.Duration, nil
}

// ScanPort scans a single port using TCP connect
func (s *ScannerService) ScanPort(ip string, port int) (*Port, error) {
	portObj := NewPort(port)
	start := time.Now()

	log.L().Debug("Scanning port", zap.String("event", "scan_port"), zap.String("ip", ip), zap.Int("port", port))

	// Try to connect with timeout
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), s.config.ConnectTimeout)
	if err != nil {
		portObj.Status = PortStatusClosed
		portObj.ResponseTime = time.Since(start)
		log.L().Debug("Port closed", zap.String("event", "port_closed"), zap.String("ip", ip), zap.Int("port", port))
		return portObj, nil // Not an error, just closed port
	}
	defer conn.Close()

	portObj.Status = PortStatusOpen
	portObj.ResponseTime = time.Since(start)
	log.L().Info("Port open", zap.String("event", "port_open"), zap.String("ip", ip), zap.Int("port", port))

	// Get banner if enabled
	if s.config.EnableBanner {
		bannerInfo, err := s.GetBanner(ip, port)
		if err == nil {
			portObj.Banner = bannerInfo.RawBanner
			portObj.Service = bannerInfo.Service
			portObj.Version = bannerInfo.Version
			portObj.BannerInfo = bannerInfo
			log.L().Info("Banner grabbed", zap.String("event", "banner_grabbed"), zap.String("ip", ip), zap.Int("port", port), zap.String("service", bannerInfo.Service), zap.String("version", bannerInfo.Version))
		} else {
			log.L().Warn("Failed to grab banner", zap.String("event", "banner_failed"), zap.String("ip", ip), zap.Int("port", port), zap.Error(err))
		}
	}

	return portObj, nil
}

// ScanPorts scans multiple ports concurrently
func (s *ScannerService) ScanPorts(ip string, ports []int) ([]*Port, error) {
	var results []*Port
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, s.config.Concurrency)

	for _, port := range ports {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Scan port with retries
			portResult, err := s.scanPortWithRetry(ip, p)
			if err != nil {
				// Log error but continue with other ports
				return
			}

			mu.Lock()
			results = append(results, portResult)
			mu.Unlock()
		}(port)
	}

	wg.Wait()

	return results, nil
}

// scanPortWithRetry scans a port with retry logic
func (s *ScannerService) scanPortWithRetry(ip string, port int) (*Port, error) {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		portResult, err := s.ScanPort(ip, port)
		if err == nil {
			return portResult, nil
		}

		lastErr = err

		if attempt < s.config.MaxRetries {
			time.Sleep(s.config.RetryDelay)
		}
	}

	return NewPort(port), lastErr
}

// GetBanner retrieves the banner from an open port using the optimized banner grabber
func (s *ScannerService) GetBanner(ip string, port int) (*BannerInfo, error) {
	// Use optimized banner grabber if available
	if s.optimizedGrabber != nil {
		return s.optimizedGrabber.GetBanner(ip, port)
	}

	// Fallback to configured banner grabber
	if s.bannerGrabber != nil {
		return s.bannerGrabber.GetBanner(ip, port)
	}

	// Fallback to basic banner grabbing
	return s.basicBannerGrab(ip, port)
}

// GetBannerStats returns banner grabbing statistics
func (s *ScannerService) GetBannerStats() map[string]interface{} {
	if s.optimizedGrabber != nil {
		return s.optimizedGrabber.GetStats()
	}
	return map[string]interface{}{
		"message": "No optimized banner grabber available",
	}
}

// Shutdown gracefully shuts down the scanner service
func (s *ScannerService) Shutdown() {
	if s.optimizedGrabber != nil {
		s.optimizedGrabber.Shutdown()
	}
}

// basicBannerGrab provides basic banner grabbing as fallback
func (s *ScannerService) basicBannerGrab(ip string, port int) (*BannerInfo, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), s.config.BannerTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(s.config.BannerTimeout))

	// Send a simple probe
	_, err = conn.Write([]byte("\r\n"))
	if err != nil {
		return nil, err
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		banner := strings.TrimSpace(scanner.Text())
		version := s.extractVersionFromBanner(banner)
		return &BannerInfo{
			RawBanner:  banner,
			Service:    s.identifyService(port, banner),
			Protocol:   "tcp",
			Version:    version,
			Confidence: "banner",
		}, nil
	}

	return &BannerInfo{
		RawBanner:  "",
		Service:    "unknown",
		Protocol:   "tcp",
		Version:    "",
		Confidence: "port",
	}, fmt.Errorf("no banner received")
}

// extractVersionFromBanner extracts version information from banner text
func (s *ScannerService) extractVersionFromBanner(banner string) string {
	// Common version patterns
	patterns := []string{
		`(?i)(?:version|v|ver)\s*[:\s]*([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
		`(?i)([0-9]+\.[0-9]+(?:\.[0-9]+)?(?:\.[0-9]+)?)`,
		`(?i)(?:openssh|ssh)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)(?:apache|nginx|iis)\s*[/\s]*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)(?:mysql|postgresql|redis|mongodb)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
		`(?i)(?:ubuntu|debian|centos|redhat|fedora)\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(banner)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}

// identifyService identifies the service based on port and banner
func (s *ScannerService) identifyService(port int, banner string) string {
	// Common port to service mappings
	portServices := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		53:   "dns",
		80:   "http",
		110:  "pop3",
		143:  "imap",
		443:  "https",
		993:  "imaps",
		995:  "pop3s",
		3306: "mysql",
		3389: "rdp",
		5432: "postgresql",
		8080: "http-proxy",
		8443: "https-alt",
	}

	if service, exists := portServices[port]; exists {
		return service
	}

	// Try to identify from banner
	bannerLower := strings.ToLower(banner)
	switch {
	case strings.Contains(bannerLower, "ssh"):
		return "ssh"
	case strings.Contains(bannerLower, "ftp"):
		return "ftp"
	case strings.Contains(bannerLower, "http"):
		return "http"
	case strings.Contains(bannerLower, "smtp"):
		return "smtp"
	case strings.Contains(bannerLower, "mysql"):
		return "mysql"
	case strings.Contains(bannerLower, "postgresql"):
		return "postgresql"
	default:
		return "unknown"
	}
}

// ScanIP performs a complete scan of an IP address
func (s *ScannerService) ScanIP(ip string, config *ScanConfig, batchID string, workerID string) (*ScanResult, error) {
	if config == nil {
		config = s.config
	}

	result := NewScanResult(ip, batchID, workerID)
	result.Status = ScanStatusRunning

	// Step 1: Ping check (if enabled)
	if config.EnablePing {
		isUp, pingTime, err := s.PingHost(ip)
		if err != nil {
			result.SetFailed(fmt.Sprintf("ping failed: %v", err))
			return result, err
		}

		result.IsUp = isUp
		result.PingTime = pingTime

		if !isUp {
			result.SetCompleted()
			return result, nil
		}
	} else {
		// Assume host is up if ping is disabled
		result.IsUp = true
	}

	// Step 2: Port scanning
	portsToScan := config.DefaultPorts
	if len(config.PortRange) > 0 {
		portsToScan = config.PortRange
	}

	ports, err := s.ScanPorts(ip, portsToScan)
	if err != nil {
		result.SetFailed(fmt.Sprintf("port scan failed: %v", err))
		return result, err
	}

	// Add ports to result
	for _, port := range ports {
		result.AddPort(port)
	}

	result.SetCompleted()
	return result, nil
}

// GetStats returns the current scan statistics
func (s *ScannerService) GetStats() *ScanStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// UpdateStats updates the scan statistics
func (s *ScannerService) UpdateStats(result *ScanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.UpdateStats(result)
}

package banner

import (
	"port-scanner/internal/domain"
	"sync"
	"time"
)

// BannerGrabber provides optimized banner grabbing with worker pool
type BannerGrabber struct {
	workerPool    *ZGrabWorkerPool
	priorityPorts map[int]bool
	timeout       time.Duration
	mu            sync.RWMutex
	stats         *BannerGrabStats
}

// BannerGrabStats tracks banner grabbing statistics
type BannerGrabStats struct {
	TotalGrabs    int64
	ZGrabGrabs    int64
	BasicGrabs    int64
	TotalDuration time.Duration
	AverageTime   time.Duration
	Errors        int64
	mu            sync.RWMutex
}

// NewBannerGrabber creates a new optimized banner grabber
func NewBannerGrabber(zgrabWorkers int, timeout time.Duration, priorityPorts []int) *BannerGrabber {
	// Create priority ports map
	priorityMap := make(map[int]bool)
	for _, port := range priorityPorts {
		priorityMap[port] = true
	}

	return &BannerGrabber{
		workerPool:    NewZGrabWorkerPool(zgrabWorkers, timeout),
		priorityPorts: priorityMap,
		timeout:       timeout,
		stats:         &BannerGrabStats{},
	}
}

// GetBanner retrieves banner information with optimization
func (o *BannerGrabber) GetBanner(ip string, port int) (*domain.BannerInfo, error) {
	start := time.Now()
	defer func() {
		o.updateStats(time.Since(start), nil)
	}()

	// Determine if this port should use ZGrab2 or basic banner grabbing
	shouldUseZGrab := o.shouldUseZGrab(port)

	if shouldUseZGrab {
		return o.getBannerWithZGrab(ip, port)
	}

	return o.getBannerBasic(ip, port)
}

// shouldUseZGrab determines if ZGrab2 should be used for this port
func (o *BannerGrabber) shouldUseZGrab(port int) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Use ZGrab2 for priority ports
	if o.priorityPorts[port] {
		return true
	}

	// Use ZGrab2 for common service ports
	commonPorts := map[int]bool{
		21: true, 22: true, 23: true, 25: true, 80: true, 110: true,
		143: true, 443: true, 993: true, 995: true, 3306: true, 5432: true,
		6379: true, 27017: true, 8080: true, 8443: true,
	}

	return commonPorts[port]
}

// getBannerWithZGrab uses ZGrab2 worker pool for banner grabbing
func (o *BannerGrabber) getBannerWithZGrab(ip string, port int) (*domain.BannerInfo, error) {
	// Determine priority based on port
	priority := o.getPortPriority(port)

	// Submit job to worker pool
	result, err := o.workerPool.SubmitJob(ip, port, priority)
	if err != nil {
		// Fallback to basic banner grabbing
		return o.getBannerBasic(ip, port)
	}

	if result.Error != nil {
		// Fallback to basic banner grabbing
		return o.getBannerBasic(ip, port)
	}

	o.updateStats(result.Duration, nil)
	return result.BannerInfo, nil
}

// getBannerBasic uses basic banner grabbing
func (o *BannerGrabber) getBannerBasic(ip string, port int) (*domain.BannerInfo, error) {
	// Use the basic banner grabber
	basicGrabber := NewZGrabBannerService(o.timeout)
	return basicGrabber.FallbackBannerGrab(ip, port)
}

// getPortPriority returns the priority for a port
func (o *BannerGrabber) getPortPriority(port int) int {
	// High priority ports
	highPriority := map[int]bool{80: true, 443: true, 22: true}
	if highPriority[port] {
		return 3
	}

	// Medium priority ports
	mediumPriority := map[int]bool{21: true, 25: true, 3306: true, 5432: true}
	if mediumPriority[port] {
		return 2
	}

	// Low priority for other ports
	return 1
}

// updateStats updates banner grabbing statistics
func (o *BannerGrabber) updateStats(duration time.Duration, err error) {
	o.stats.mu.Lock()
	defer o.stats.mu.Unlock()

	o.stats.TotalGrabs++
	o.stats.TotalDuration += duration

	if err != nil {
		o.stats.Errors++
	}

	// Update average time
	if o.stats.TotalGrabs > 0 {
		o.stats.AverageTime = o.stats.TotalDuration / time.Duration(o.stats.TotalGrabs)
	}
}

// GetStats returns banner grabbing statistics
func (o *BannerGrabber) GetStats() map[string]interface{} {
	o.stats.mu.RLock()
	defer o.stats.mu.RUnlock()

	poolStats := o.workerPool.GetStats()

	return map[string]interface{}{
		"total_grabs":    o.stats.TotalGrabs,
		"zgrab_grabs":    o.stats.ZGrabGrabs,
		"basic_grabs":    o.stats.BasicGrabs,
		"total_duration": o.stats.TotalDuration.String(),
		"average_time":   o.stats.AverageTime.String(),
		"errors":         o.stats.Errors,
		"error_rate":     float64(o.stats.Errors) / float64(o.stats.TotalGrabs) * 100,
		"worker_pool":    poolStats,
	}
}

// Shutdown gracefully shuts down the banner grabber
func (o *BannerGrabber) Shutdown() {
	o.workerPool.Shutdown()
}

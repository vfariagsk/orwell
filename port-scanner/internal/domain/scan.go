package domain

import (
	"time"
)

// ScanStatus represents the status of a scan operation
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusTimeout   ScanStatus = "timeout"
)

// PortStatus represents the status of a port
type PortStatus string

const (
	PortStatusOpen     PortStatus = "open"
	PortStatusClosed   PortStatus = "closed"
	PortStatusFiltered PortStatus = "filtered"
)

// IPAddress represents an IPv4 address to be scanned
type IPAddress struct {
	Address string
	Status  ScanStatus
}

// NewIPAddress creates a new IP address
func NewIPAddress(address string) *IPAddress {
	return &IPAddress{
		Address: address,
		Status:  ScanStatusPending,
	}
}

// BannerInfo represents comprehensive banner information
type BannerInfo struct {
	RawBanner  string                 `json:"raw_banner"`
	Service    string                 `json:"service"`
	Protocol   string                 `json:"protocol"`
	Version    string                 `json:"version,omitempty"`
	Confidence string                 `json:"confidence"` // "banner", "port", "zgrab2"
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Port represents a network port
type Port struct {
	Number       int
	Status       PortStatus
	Service      string
	Banner       string
	Version      string
	ScanTime     time.Time
	ResponseTime time.Duration
	BannerInfo   *BannerInfo `json:"banner_info,omitempty"`
}

// NewPort creates a new port
func NewPort(number int) *Port {
	return &Port{
		Number:   number,
		Status:   PortStatusClosed,
		ScanTime: time.Now(),
	}
}

// ScanResult represents the complete scan result for an IP
type ScanResult struct {
	IP            string
	IsUp          bool
	PingTime      time.Duration
	ScanStartTime time.Time
	ScanEndTime   time.Time
	Ports         []*Port
	Status        ScanStatus
	Error         string
	BatchID       string
	WorkerID      string
}

// NewScanResult creates a new scan result
func NewScanResult(ip string, batchID string, workerID string) *ScanResult {
	return &ScanResult{
		IP:            ip,
		ScanStartTime: time.Now(),
		Status:        ScanStatusPending,
		BatchID:       batchID,
		Ports:         make([]*Port, 0),
		WorkerID:      workerID,
	}
}

// AddPort adds a port to the scan result
func (sr *ScanResult) AddPort(port *Port) {
	sr.Ports = append(sr.Ports, port)
}

// SetCompleted marks the scan as completed
func (sr *ScanResult) SetCompleted() {
	sr.ScanEndTime = time.Now()
	sr.Status = ScanStatusCompleted
}

// SetFailed marks the scan as failed
func (sr *ScanResult) SetFailed(err string) {
	sr.ScanEndTime = time.Now()
	sr.Status = ScanStatusFailed
	sr.Error = err
}

// GetOpenPorts returns all open ports
func (sr *ScanResult) GetOpenPorts() []*Port {
	var openPorts []*Port
	for _, port := range sr.Ports {
		if port.Status == PortStatusOpen {
			openPorts = append(openPorts, port)
		}
	}
	return openPorts
}

// GetScanDuration returns the total scan duration
func (sr *ScanResult) GetScanDuration() time.Duration {
	if sr.ScanEndTime.IsZero() {
		return time.Since(sr.ScanStartTime)
	}
	return sr.ScanEndTime.Sub(sr.ScanStartTime)
}

// ScanConfig represents configuration for scanning
type ScanConfig struct {
	PingTimeout      time.Duration
	ConnectTimeout   time.Duration
	BannerTimeout    time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	Concurrency      int
	ZGrabConcurrency int // Maximum concurrent ZGrab2 processes
	PortRange        []int
	DefaultPorts     []int
	EnableBanner     bool
	EnablePing       bool
	PriorityPorts    []int // Ports that should get priority for banner grabbing
}

// NewDefaultScanConfig creates a default scan configuration
func NewDefaultScanConfig() *ScanConfig {
	return &ScanConfig{
		PingTimeout:      5 * time.Second,
		ConnectTimeout:   3 * time.Second,
		BannerTimeout:    2 * time.Second,
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		Concurrency:      100,
		ZGrabConcurrency: 20, // Limit ZGrab2 processes to avoid system overload
		DefaultPorts:     []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995, 3306, 3389, 5432, 8080, 8443},
		PriorityPorts:    []int{80, 443, 22, 21, 25, 3306, 5432}, // High-priority ports for banner grabbing
		EnableBanner:     true,
		EnablePing:       true,
	}
}

// BannerGrabber defines the interface for banner grabbing operations
type BannerGrabber interface {
	GetBanner(ip string, port int) (*BannerInfo, error)
}

// OptimizedBannerGrabber defines the interface for optimized banner grabbing operations
type OptimizedBannerGrabber interface {
	GetBanner(ip string, port int) (*BannerInfo, error)
	GetStats() map[string]interface{}
	Shutdown()
}

// Scanner defines the interface for port scanning operations
type Scanner interface {
	PingHost(ip string) (bool, time.Duration, error)
	ScanPort(ip string, port int) (*Port, error)
	ScanPorts(ip string, ports []int) ([]*Port, error)
	GetBanner(ip string, port int) (*BannerInfo, error)
	ScanIP(ip string, config *ScanConfig, batchID string, workerID string) (*ScanResult, error)
}

// ScanEngine defines the interface for the main scanning engine
type ScanEngine interface {
	StartScanning()
	StopScanning()
	ProcessIP(ip string, batchID string) error
	GetScanStatus(ip string) (*ScanResult, error)
	GetScanStats() *ScanStats
}

// ScanStats represents scanning statistics
type ScanStats struct {
	TotalScanned    int64
	SuccessfulScans int64
	FailedScans     int64
	AverageScanTime time.Duration
	StartTime       time.Time
	LastScanTime    time.Time
}

// NewScanStats creates new scan statistics
func NewScanStats() *ScanStats {
	return &ScanStats{
		StartTime: time.Now(),
	}
}

// UpdateStats updates the scan statistics
func (ss *ScanStats) UpdateStats(result *ScanResult) {
	ss.TotalScanned++
	ss.LastScanTime = time.Now()

	if result.Status == ScanStatusCompleted {
		ss.SuccessfulScans++
	} else {
		ss.FailedScans++
	}

	// Update average scan time
	if ss.SuccessfulScans > 0 {
		totalTime := ss.AverageScanTime * time.Duration(ss.SuccessfulScans-1)
		totalTime += result.GetScanDuration()
		ss.AverageScanTime = totalTime / time.Duration(ss.SuccessfulScans)
	}
}

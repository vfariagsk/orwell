package domain

// QueueMessage represents a message from the IP generator queue
type QueueMessage struct {
	IPs     []string `json:"ips"`
	BatchID string   `json:"batch_id"`
	Count   int      `json:"count"`
}

// ScanResultMessage represents a scan result message for output queues
type ScanResultMessage struct {
	ScanResult *ScanResult `json:"scan_result"`
	Timestamp  int64       `json:"timestamp"`
	WorkerID   string      `json:"worker_id"`
}

// EnrichmentMessage represents a message for IP enrichment queue
type EnrichmentMessage struct {
	IP        string `json:"ip"`
	IsUp      bool   `json:"is_up"`
	BatchID   string `json:"batch_id"`
	Timestamp int64  `json:"timestamp"`
}

// ServiceAnalysisMessage represents a message for service analysis queue
type ServiceAnalysisMessage struct {
	IP        string  `json:"ip"`
	OpenPorts []*Port `json:"open_ports"`
	BatchID   string  `json:"batch_id"`
	Timestamp int64   `json:"timestamp"`
}

// QueueConsumer defines the interface for consuming messages from queues
type QueueConsumer interface {
	Consume(handler func(*QueueMessage) error) error
	Stop() error
}

// QueuePublisher defines the interface for publishing messages to queues
type QueuePublisher interface {
	Publish(message interface{}) error
	PublishBatch(messages []interface{}) error
}

// QueueManager defines the interface for managing multiple queues
type QueueManager interface {
	ConsumeIPs(handler func(*QueueMessage) error) error
	PublishScanResult(result *ScanResult) error
	PublishEnrichmentMessage(ip string, isUp bool, batchID string) error
	PublishServiceAnalysis(ip string, openPorts []*Port, batchID string) error
	Close() error
}

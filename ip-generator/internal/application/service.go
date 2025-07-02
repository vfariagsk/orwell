package application

import (
	"fmt"
	"net"
	"time"

	"ip-generator/internal/domain"
)

// IPGenerationService handles the business logic for IP generation and queue publishing
type IPGenerationService struct {
	ipGenerator    domain.IPGenerator
	queuePublisher domain.QueuePublisher
}

// NewIPGenerationService creates a new IP generation service
func NewIPGenerationService(ipGenerator domain.IPGenerator, queuePublisher domain.QueuePublisher) *IPGenerationService {
	return &IPGenerationService{
		ipGenerator:    ipGenerator,
		queuePublisher: queuePublisher,
	}
}

// GenerateAndPublishIPs generates IPs and publishes them to the queue
func (s *IPGenerationService) GenerateAndPublishIPs(count int, batchSize int) error {
	if count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	if batchSize <= 0 {
		batchSize = 100 // default batch size
	}

	// Calculate number of batches
	numBatches := (count + batchSize - 1) / batchSize
	batchID := generateBatchID()

	var messages []*domain.QueueMessage

	for i := 0; i < numBatches; i++ {
		currentBatchSize := batchSize
		if i == numBatches-1 && count%batchSize != 0 {
			currentBatchSize = count % batchSize
		}

		// Generate IPs for this batch
		ips, err := s.ipGenerator.GenerateIPs(currentBatchSize)
		if err != nil {
			return fmt.Errorf("failed to generate IPs for batch %d: %w", i, err)
		}

		// Convert domain IPs to strings
		ipStrings := make([]string, len(ips))
		for j, ip := range ips {
			ipStrings[j] = ip.String()
		}

		// Create queue message
		message := &domain.QueueMessage{
			IPs:     ipStrings,
			BatchID: fmt.Sprintf("%s-%d", batchID, i),
			Count:   len(ipStrings),
		}

		messages = append(messages, message)
	}

	// Publish all messages to queue
	if err := s.queuePublisher.PublishBatch(messages); err != nil {
		return fmt.Errorf("failed to publish messages to queue: %w", err)
	}

	return nil
}

// GenerateAndPublishSequentialIPs generates sequential IPs and publishes them to the queue
func (s *IPGenerationService) GenerateAndPublishSequentialIPs(startIP string, count int, batchSize int) error {
	if count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	if batchSize <= 0 {
		batchSize = 100 // default batch size
	}

	// Calculate number of batches
	numBatches := (count + batchSize - 1) / batchSize
	batchID := generateBatchID()

	var messages []*domain.QueueMessage
	currentStartIP := startIP

	for i := 0; i < numBatches; i++ {
		currentBatchSize := batchSize
		if i == numBatches-1 && count%batchSize != 0 {
			currentBatchSize = count % batchSize
		}

		// Generate sequential IPs for this batch
		ips, err := s.ipGenerator.GenerateSequentialIPs(currentStartIP, currentBatchSize)
		if err != nil {
			return fmt.Errorf("failed to generate sequential IPs for batch %d: %w", i, err)
		}

		// Convert domain IPs to strings
		ipStrings := make([]string, len(ips))
		for j, ip := range ips {
			ipStrings[j] = ip.String()
		}

		// Create queue message
		message := &domain.QueueMessage{
			IPs:     ipStrings,
			BatchID: fmt.Sprintf("%s-%d", batchID, i),
			Count:   len(ipStrings),
		}

		messages = append(messages, message)

		// Update start IP for next batch
		if len(ips) > 0 {
			currentStartIP = incrementIP(ips[len(ips)-1].String())
		}
	}

	// Publish all messages to queue
	if err := s.queuePublisher.PublishBatch(messages); err != nil {
		return fmt.Errorf("failed to publish messages to queue: %w", err)
	}

	return nil
}

// generateBatchID generates a unique batch ID
func generateBatchID() string {
	return fmt.Sprintf("batch-%d", time.Now().UnixNano())
}

// incrementIP increments an IP address by 1
func incrementIP(ipStr string) string {
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return ipStr
	}

	ip := parsedIP.To4()
	if ip == nil {
		return ipStr
	}

	// Increment IP address
	for j := 3; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}

	return ip.String()
}

package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"port-scanner/internal/domain"
	"port-scanner/internal/infrastructure/database"
	"port-scanner/pkg/log"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// RabbitMQManager implements the QueueManager interface
type RabbitMQManager struct {
	conn                 *amqp.Connection
	channel              *amqp.Channel
	ipQueue              string
	scanResultQueue      string
	enrichmentQueue      string
	serviceAnalysisQueue string
	workerID             string
	scanHandler          func(string, *domain.ScanConfig, string, string) (*domain.ScanResult, error)
	scanConfig           *domain.ScanConfig
	dbManager            *database.MongoDBManager
}

// NewRabbitMQManager creates a new RabbitMQ manager
func NewRabbitMQManager(url, ipQueue, scanResultQueue, enrichmentQueue, serviceAnalysisQueue string) (*RabbitMQManager, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare all queues
	queues := []string{ipQueue, scanResultQueue, enrichmentQueue, serviceAnalysisQueue}
	for _, queueName := range queues {
		_, err = ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}
	}

	// Generate worker ID
	workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())

	return &RabbitMQManager{
		conn:                 conn,
		channel:              ch,
		ipQueue:              ipQueue,
		scanResultQueue:      scanResultQueue,
		enrichmentQueue:      enrichmentQueue,
		serviceAnalysisQueue: serviceAnalysisQueue,
		workerID:             workerID,
		scanConfig:           domain.NewDefaultScanConfig(),
	}, nil
}

// SetMongoDBManager sets the MongoDB manager for saving results
func (r *RabbitMQManager) SetMongoDBManager(dbManager *database.MongoDBManager) {
	r.dbManager = dbManager
}

// SetScanHandler sets the scan handler function
func (r *RabbitMQManager) SetScanHandler(handler func(string, *domain.ScanConfig, string, string) (*domain.ScanResult, error)) {
	r.scanHandler = handler
}

// SetScanConfig sets the scan configuration
func (r *RabbitMQManager) SetScanConfig(config *domain.ScanConfig) {
	r.scanConfig = config
}

// ConsumeIPs starts consuming IP messages from the queue
func (r *RabbitMQManager) ConsumeIPs(handler func(*domain.QueueMessage) error) error {
	msgs, err := r.channel.Consume(
		r.ipQueue, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for msg := range msgs {
			err := r.handleMessage(msg)
			if err != nil {
				log.L().Error("Failed to process message", zap.String("event", "process_failed"), zap.Error(err))
				msg.Nack(false, true) // requeue
			}
		}
	}()

	return nil
}

func (r *RabbitMQManager) handleMessage(delivery amqp.Delivery) error {
	var message domain.QueueMessage
	err := json.Unmarshal(delivery.Body, &message)
	if err != nil {
		log.L().Error("Failed to unmarshal message", zap.String("event", "unmarshal_failed"), zap.Error(err))
		return err
	}

	log.L().Info("Processing IP message", zap.String("event", "ip_processing_started"),
		zap.Strings("ips", message.IPs), zap.String("batch_id", message.BatchID))

	// Validate IP addresses
	if len(message.IPs) == 0 {
		log.L().Error("Invalid message: no IP addresses", zap.String("event", "invalid_message"))
		delivery.Ack(false) // Don't requeue invalid messages
		return fmt.Errorf("no IP addresses in message")
	}

	// Check if scan handler is set
	if r.scanHandler == nil {
		log.L().Error("Scan handler not set", zap.String("event", "handler_not_set"))
		return fmt.Errorf("scan handler not configured")
	}

	// Process each IP in the message
	for _, ip := range message.IPs {
		if ip == "" {
			log.L().Warn("Skipping empty IP address", zap.String("event", "empty_ip_skipped"))
			continue
		}

		// Perform the scan
		startTime := time.Now()
		result, err := r.scanHandler(ip, r.scanConfig, message.BatchID, r.workerID)
		scanDuration := time.Since(startTime)

		if err != nil {
			log.L().Error("Scan failed", zap.String("event", "scan_failed"),
				zap.String("ip", ip), zap.Error(err), zap.Duration("duration", scanDuration))

			// Create a failed result for tracking
			failedResult := &domain.ScanResult{
				IP:            ip,
				Status:        domain.ScanStatusFailed,
				IsUp:          false,
				Error:         err.Error(),
				ScanStartTime: startTime,
				ScanEndTime:   time.Now(),
				BatchID:       message.BatchID,
				WorkerID:      r.workerID,
			}

			// Save to MongoDB if available
			if r.dbManager != nil {
				if saveErr := r.dbManager.SaveScanResult(failedResult); saveErr != nil {
					log.L().Error("Failed to save failed result to MongoDB", zap.String("event", "mongodb_save_failed"), zap.Error(saveErr))
				}
			}

			// Publish failed result
			if pubErr := r.PublishScanResult(failedResult); pubErr != nil {
				log.L().Error("Failed to publish failed result", zap.String("event", "publish_failed"), zap.Error(pubErr))
			}

			// Publish enrichment message for failed scan
			if pubErr := r.PublishEnrichmentMessage(ip, false, message.BatchID); pubErr != nil {
				log.L().Error("Failed to publish enrichment message", zap.String("event", "enrichment_failed"), zap.Error(pubErr))
			}

			continue // Continue with next IP
		}

		// Set batch ID if not set
		if result.BatchID == "" {
			result.BatchID = message.BatchID
		}

		log.L().Info("Scan completed successfully", zap.String("event", "scan_completed"),
			zap.String("ip", result.IP), zap.Bool("is_up", result.IsUp),
			zap.Int("open_ports", len(result.GetOpenPorts())), zap.Duration("duration", scanDuration))

		// Save to MongoDB if available
		if r.dbManager != nil {
			if saveErr := r.dbManager.SaveScanResult(result); saveErr != nil {
				log.L().Error("Failed to save scan result to MongoDB", zap.String("event", "mongodb_save_failed"), zap.Error(saveErr))
			}
		}

		// Publish scan result
		if err := r.PublishScanResult(result); err != nil {
			log.L().Error("Failed to publish scan result", zap.String("event", "publish_failed"), zap.Error(err))
			continue
		}

		// Publish enrichment message
		if err := r.PublishEnrichmentMessage(result.IP, result.IsUp, result.BatchID); err != nil {
			log.L().Error("Failed to publish enrichment message", zap.String("event", "enrichment_failed"), zap.Error(err))
		}

		// Publish service analysis if there are open ports
		openPorts := result.GetOpenPorts()
		if len(openPorts) > 0 {
			if err := r.PublishServiceAnalysis(result.IP, openPorts, result.BatchID); err != nil {
				log.L().Error("Failed to publish service analysis", zap.String("event", "service_analysis_failed"), zap.Error(err))
			}
		}
	}

	// Acknowledge the message
	delivery.Ack(false)
	return nil
}

// PublishScanResult publishes a scan result to the scan result queue
func (r *RabbitMQManager) PublishScanResult(result *domain.ScanResult) error {
	message := domain.ScanResultMessage{
		ScanResult: result,
		Timestamp:  time.Now().Unix(),
		WorkerID:   "port-scanner",
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = r.channel.Publish(
		"",
		r.scanResultQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return err
	}

	log.L().Info("Published scan result", zap.String("event", "scan_result_published"), zap.String("ip", result.IP))
	return nil
}

// PublishEnrichmentMessage publishes an enrichment message
func (r *RabbitMQManager) PublishEnrichmentMessage(ip string, isUp bool, batchID string) error {
	message := domain.EnrichmentMessage{
		IP:        ip,
		IsUp:      isUp,
		BatchID:   batchID,
		Timestamp: time.Now().Unix(),
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = r.channel.Publish(
		"",
		r.enrichmentQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return err
	}

	log.L().Info("Published enrichment message", zap.String("event", "enrichment_published"), zap.String("ip", ip), zap.Bool("is_up", isUp))
	return nil
}

// PublishServiceAnalysis publishes a service analysis message
func (r *RabbitMQManager) PublishServiceAnalysis(ip string, openPorts []*domain.Port, batchID string) error {
	message := domain.ServiceAnalysisMessage{
		IP:        ip,
		OpenPorts: openPorts,
		BatchID:   batchID,
		Timestamp: time.Now().Unix(),
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = r.channel.Publish(
		"",
		r.serviceAnalysisQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return err
	}

	log.L().Info("Published service analysis", zap.String("event", "service_analysis_published"), zap.String("ip", ip), zap.Int("open_ports", len(openPorts)))
	return nil
}

// Close closes the RabbitMQ connection
func (r *RabbitMQManager) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}

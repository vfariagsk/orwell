package queue

import (
	"encoding/json"
	"fmt"

	"ip-generator/internal/domain"
	"ip-generator/pkg/log"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// RabbitMQPublisher implements the QueuePublisher interface using RabbitMQ
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(url, queueName string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the queue
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
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
}

// Publish publishes a single message to the queue
func (r *RabbitMQPublisher) Publish(message *domain.QueueMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.Publish(
		"",      // exchange
		r.queue, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.L().Info("Published message", zap.String("event", "message_published"), zap.String("batch_id", message.BatchID), zap.Int("ip_count", message.Count))
	return nil
}

// PublishBatch publishes multiple messages to the queue
func (r *RabbitMQPublisher) PublishBatch(messages []*domain.QueueMessage) error {
	for _, message := range messages {
		if err := r.Publish(message); err != nil {
			return fmt.Errorf("failed to publish message in batch: %w", err)
		}
	}
	return nil
}

// Close closes the RabbitMQ connection
func (r *RabbitMQPublisher) Close() error {
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

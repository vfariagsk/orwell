package domain

// QueueMessage represents a message to be sent to the queue
type QueueMessage struct {
	IPs     []string `json:"ips"`
	BatchID string   `json:"batch_id"`
	Count   int      `json:"count"`
}

// QueuePublisher defines the interface for publishing messages to a queue
type QueuePublisher interface {
	Publish(message *QueueMessage) error
	PublishBatch(messages []*QueueMessage) error
}

// QueueSubscriber defines the interface for subscribing to queue messages
type QueueSubscriber interface {
	Subscribe(handler func(*QueueMessage) error) error
	Unsubscribe() error
}

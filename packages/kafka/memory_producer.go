package kafka

import (
	"context"
	"sync"
)

type MemoryProducer struct {
	mu       sync.RWMutex
	messages []Message
}

func NewMemoryProducer() *MemoryProducer {
	return &MemoryProducer{}
}

func (producer *MemoryProducer) Publish(ctx context.Context, message Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	producer.mu.Lock()
	defer producer.mu.Unlock()
	producer.messages = append(producer.messages, message)
	return nil
}

func (producer *MemoryProducer) Messages() []Message {
	producer.mu.RLock()
	defer producer.mu.RUnlock()
	return append([]Message(nil), producer.messages...)
}


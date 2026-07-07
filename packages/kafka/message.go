package kafka

import "context"

type Message struct {
	Topic string
	Key   string
	Value []byte
}

type Producer interface {
	Publish(ctx context.Context, message Message) error
}


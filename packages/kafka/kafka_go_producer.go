package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type WriterProducer struct {
	writer *kafkago.Writer
}

func NewWriterProducer(brokers []string) *WriterProducer {
	return &WriterProducer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireOne,
			BatchTimeout: 50 * time.Millisecond,
		},
	}
}

func (producer *WriterProducer) Publish(ctx context.Context, message Message) error {
	return producer.writer.WriteMessages(ctx, kafkago.Message{
		Topic: message.Topic,
		Key:   []byte(message.Key),
		Value: message.Value,
		Time:  time.Now().UTC(),
	})
}

func (producer *WriterProducer) Close() error {
	return producer.writer.Close()
}


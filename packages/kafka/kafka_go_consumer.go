package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
)

type Consumer interface {
	Fetch(ctx context.Context) (Message, error)
	Close() error
}

type ReaderConsumer struct {
	reader *kafkago.Reader
}

func NewReaderConsumer(brokers []string, topic string, groupID string) *ReaderConsumer {
	return &ReaderConsumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

func (consumer *ReaderConsumer) Fetch(ctx context.Context) (Message, error) {
	message, err := consumer.reader.FetchMessage(ctx)
	if err != nil {
		return Message{}, err
	}
	return FromKafkaGoMessage(message), nil
}

func (consumer *ReaderConsumer) Commit(ctx context.Context, message Message) error {
	return consumer.reader.CommitMessages(ctx, kafkago.Message{
		Topic: message.Topic,
		Key:   []byte(message.Key),
		Value: message.Value,
	})
}

func (consumer *ReaderConsumer) Close() error {
	return consumer.reader.Close()
}

func FromKafkaGoMessage(message kafkago.Message) Message {
	return Message{
		Topic: message.Topic,
		Key:   string(message.Key),
		Value: append([]byte(nil), message.Value...),
	}
}


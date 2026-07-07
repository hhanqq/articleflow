package kafka

import (
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestFromKafkaGoMessage(t *testing.T) {
	source := kafkago.Message{
		Topic: "article.discovered.v1",
		Key:   []byte("habr-123"),
		Value: []byte(`{"title":"Go"}`),
		Time:  time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
	}

	message := FromKafkaGoMessage(source)

	if message.Topic != source.Topic {
		t.Fatalf("expected topic %s, got %s", source.Topic, message.Topic)
	}
	if message.Key != "habr-123" {
		t.Fatalf("unexpected key: %s", message.Key)
	}
	if string(message.Value) != `{"title":"Go"}` {
		t.Fatalf("unexpected value: %s", string(message.Value))
	}
}


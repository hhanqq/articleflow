package kafka

import (
	"context"
	"testing"
)

func TestMemoryProducerStoresPublishedMessages(t *testing.T) {
	producer := NewMemoryProducer()
	message := Message{
		Topic: "article.discovered.v1",
		Key:   "article-1",
		Value: []byte(`{"id":"article-1"}`),
	}

	if err := producer.Publish(context.Background(), message); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Topic != message.Topic {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
}


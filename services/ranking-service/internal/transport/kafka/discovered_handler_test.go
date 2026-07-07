package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/usecase"
)

func TestDiscoveredHandlerPublishesScoredFeedItem(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	handler := NewDiscoveredHandler(usecase.NewRanker(), producer)
	event := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go Kafka",
		Summary:      "Streaming microservices",
		Tags:         []string{"go", "kafka"},
		PublishedAt:  time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Now().UTC(),
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	err = handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   event.ExternalID,
		Value: payload,
	})

	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicFeedItemScored {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	var scored eventsv1.FeedItemScoredEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &scored); err != nil {
		t.Fatalf("unmarshal scored event: %v", err)
	}
	if scored.ArticleID != "habr:123" {
		t.Fatalf("unexpected article id: %s", scored.ArticleID)
	}
	if scored.Score <= 0 {
		t.Fatalf("expected positive score, got %f", scored.Score)
	}
}

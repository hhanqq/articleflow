package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

func TestDiscoveredHandlerIngestsKafkaMessage(t *testing.T) {
	store := usecase.NewMemoryArticleStore()
	ingest := usecase.NewIngestUsecase(store)
	handler := NewDiscoveredHandler(ingest)
	event := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-123",
		URL:          "https://habr.com/ru/articles/123/",
		Title:        "Go microservices",
		DiscoveredAt: time.Now().UTC(),
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	created, err := handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   event.ExternalID,
		Value: payload,
	})

	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	if created.ArticleID == "" {
		t.Fatal("expected created article id")
	}
	if _, ok := store.FindByURL(event.URL); !ok {
		t.Fatal("expected article stored after handler")
	}
}

func TestDiscoveredHandlerRejectsWrongTopic(t *testing.T) {
	handler := NewDiscoveredHandler(usecase.NewIngestUsecase(usecase.NewMemoryArticleStore()))

	_, err := handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: "unknown.v1",
		Value: []byte(`{}`),
	})

	if err == nil {
		t.Fatal("expected error for wrong topic")
	}
}


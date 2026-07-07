package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

func TestRuntimeMessageHandlerDelegatesToDiscoveredHandler(t *testing.T) {
	payload, err := articleflowkafka.MarshalJSON(eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-1",
		URL:          "https://habr.com/ru/articles/1/",
		Title:        "Go Kafka",
		DiscoveredAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	store := usecase.NewMemoryArticleStore()
	ingest := usecase.NewIngestUsecase(store)
	handler := NewRuntimeMessageHandler(NewDiscoveredHandler(ingest))

	err = handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   "habr-1",
		Value: payload,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.FindByURL("https://habr.com/ru/articles/1/"); !ok {
		t.Fatalf("expected article to be ingested")
	}
}

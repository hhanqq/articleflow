package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/feed-service/internal/usecase"
)

func TestScoredHandlerStoresFeedItem(t *testing.T) {
	feed := usecase.NewMemoryFeed()
	handler := NewScoredHandler(feed)
	event := eventsv1.FeedItemScoredEvent{
		ArticleID:   "habr:123",
		SourceName:  "habr",
		URL:         "https://habr.com/ru/articles/123/",
		Title:       "Go Kafka",
		Score:       55,
		PublishedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		ScoredAt:    time.Now().UTC(),
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	err = handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicFeedItemScored,
		Key:   event.ArticleID,
		Value: payload,
	})

	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}
	items, err := feed.List(10)
	if err != nil {
		t.Fatalf("list feed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(items))
	}
	if items[0].Score != 55 {
		t.Fatalf("unexpected score: %f", items[0].Score)
	}
}

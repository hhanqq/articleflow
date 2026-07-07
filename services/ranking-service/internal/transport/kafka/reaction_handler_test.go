package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/usecase"
)

func TestReactionHandlerUpdatesRankingSignals(t *testing.T) {
	signals := usecase.NewSignalStore()
	signals.RememberArticle(feedv1.FeedItem{
		ArticleID: "habr:123",
		Title:     "Go Kafka",
		Tags:      []string{"go", "kafka"},
	})
	handler := NewReactionHandler(signals)
	event := eventsv1.UserReactionCreatedEvent{
		UserID:    "reader-1",
		ArticleID: "habr:123",
		Type:      string(userv1.ReactionLike),
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	err = handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicUserReactionCreated,
		Key:   "reader-1:habr:123",
		Value: payload,
	})

	if err != nil {
		t.Fatalf("handle reaction event: %v", err)
	}
	score := signals.ApplyToScore(10, feedv1.FeedItem{
		ArticleID: "habr:456",
		Title:     "Go routines",
		Tags:      []string{"go"},
	})
	if score <= 10 {
		t.Fatalf("expected reaction signal boost, got %f", score)
	}
}

package kafka

import (
	"context"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/user-service/internal/usecase"
)

func TestReactionHandlerStoresReactionEvent(t *testing.T) {
	store := usecase.NewMemoryReactions()
	handler := NewReactionHandler(store)
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
	reactions := store.ListByUser("reader-1")
	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}
	if reactions[0].Type != userv1.ReactionLike {
		t.Fatalf("unexpected reaction type: %s", reactions[0].Type)
	}
}

func TestReactionHandlerRejectsUnexpectedTopic(t *testing.T) {
	handler := NewReactionHandler(usecase.NewMemoryReactions())

	err := handler.Handle(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   "article-1",
		Value: []byte(`{}`),
	})

	if err == nil {
		t.Fatal("expected unexpected topic error")
	}
}

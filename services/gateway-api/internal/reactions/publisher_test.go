package reactions

import (
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

func TestPublisherPublishesUserReactionCreatedEvent(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	publisher := NewPublisher(producer)

	err := publisher.Record(userv1.UserReaction{
		UserID:    "reader-1",
		ArticleID: "habr:1",
		Type:      userv1.ReactionSave,
		CreatedAt: time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("record reaction: %v", err)
	}
	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicUserReactionCreated {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	var event eventsv1.UserReactionCreatedEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.UserID != "reader-1" || event.ArticleID != "habr:1" || event.Type != "save" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

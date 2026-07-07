package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/feed-service/internal/usecase"
)

type ScoredHandler struct {
	feed *usecase.MemoryFeed
}

func NewScoredHandler(feed *usecase.MemoryFeed) *ScoredHandler {
	return &ScoredHandler{feed: feed}
}

func (handler *ScoredHandler) Handle(_ context.Context, message articleflowkafka.Message) error {
	if message.Topic != eventsv1.TopicFeedItemScored {
		return errors.New("unexpected kafka topic")
	}
	var event eventsv1.FeedItemScoredEvent
	if err := articleflowkafka.UnmarshalJSON(message.Value, &event); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}
	handler.feed.UpsertScoredItem(event)
	return nil
}

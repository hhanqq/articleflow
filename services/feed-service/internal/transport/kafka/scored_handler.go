package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type ScoredFeedStore interface {
	UpsertScoredItem(event eventsv1.FeedItemScoredEvent) error
}

type ScoredHandler struct {
	feed ScoredFeedStore
}

func NewScoredHandler(feed ScoredFeedStore) *ScoredHandler {
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
	return handler.feed.UpsertScoredItem(event)
}

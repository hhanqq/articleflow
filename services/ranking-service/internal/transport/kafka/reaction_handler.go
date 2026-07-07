package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/usecase"
)

type ReactionHandler struct {
	signals *usecase.SignalStore
}

func NewReactionHandler(signals *usecase.SignalStore) *ReactionHandler {
	return &ReactionHandler{signals: signals}
}

func (handler *ReactionHandler) Handle(_ context.Context, message articleflowkafka.Message) error {
	if message.Topic != eventsv1.TopicUserReactionCreated {
		return errors.New("unexpected kafka topic")
	}

	var event eventsv1.UserReactionCreatedEvent
	if err := articleflowkafka.UnmarshalJSON(message.Value, &event); err != nil {
		return err
	}
	return handler.signals.RecordReaction(event)
}

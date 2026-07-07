package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type ReactionRecorder interface {
	Record(reaction userv1.UserReaction) error
}

type ReactionHandler struct {
	recorder ReactionRecorder
}

func NewReactionHandler(recorder ReactionRecorder) *ReactionHandler {
	return &ReactionHandler{recorder: recorder}
}

func (handler *ReactionHandler) Handle(_ context.Context, message articleflowkafka.Message) error {
	if message.Topic != eventsv1.TopicUserReactionCreated {
		return errors.New("unexpected kafka topic")
	}

	var event eventsv1.UserReactionCreatedEvent
	if err := articleflowkafka.UnmarshalJSON(message.Value, &event); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}

	return handler.recorder.Record(userv1.UserReaction{
		UserID:    event.UserID,
		ArticleID: event.ArticleID,
		Type:      userv1.ReactionType(event.Type),
		CreatedAt: event.CreatedAt,
	})
}

package reactions

import (
	"context"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type Publisher struct {
	producer articleflowkafka.Producer
}

func NewPublisher(producer articleflowkafka.Producer) *Publisher {
	return &Publisher{producer: producer}
}

func (publisher *Publisher) Record(reaction userv1.UserReaction) error {
	event := eventsv1.UserReactionCreatedEvent{
		UserID:    reaction.UserID,
		ArticleID: reaction.ArticleID,
		Type:      string(reaction.Type),
		CreatedAt: reaction.CreatedAt,
	}
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		return err
	}
	return publisher.producer.Publish(context.Background(), articleflowkafka.Message{
		Topic: eventsv1.TopicUserReactionCreated,
		Key:   event.UserID + ":" + event.ArticleID,
		Value: payload,
	})
}

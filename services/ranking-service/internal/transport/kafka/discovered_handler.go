package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/usecase"
)

type DiscoveredHandler struct {
	ranker   *usecase.Ranker
	producer articleflowkafka.Producer
}

func NewDiscoveredHandler(ranker *usecase.Ranker, producer articleflowkafka.Producer) *DiscoveredHandler {
	return &DiscoveredHandler{ranker: ranker, producer: producer}
}

func (handler *DiscoveredHandler) Handle(ctx context.Context, message articleflowkafka.Message) error {
	if message.Topic != eventsv1.TopicArticleDiscovered {
		return errors.New("unexpected kafka topic")
	}
	var event eventsv1.ArticleDiscoveredEvent
	if err := articleflowkafka.UnmarshalJSON(message.Value, &event); err != nil {
		return err
	}
	scored := handler.ranker.RankDiscovered(event)
	if err := scored.Validate(); err != nil {
		return err
	}
	payload, err := articleflowkafka.MarshalJSON(scored)
	if err != nil {
		return err
	}
	return handler.producer.Publish(ctx, articleflowkafka.Message{
		Topic: eventsv1.TopicFeedItemScored,
		Key:   scored.ArticleID,
		Value: payload,
	})
}

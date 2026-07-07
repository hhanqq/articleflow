package kafka

import (
	"context"
	"errors"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

type DiscoveredHandler struct {
	ingest *usecase.IngestUsecase
}

func NewDiscoveredHandler(ingest *usecase.IngestUsecase) *DiscoveredHandler {
	return &DiscoveredHandler{ingest: ingest}
}

func (handler *DiscoveredHandler) Handle(ctx context.Context, message articleflowkafka.Message) (eventsv1.ArticleCreatedEvent, error) {
	if message.Topic != eventsv1.TopicArticleDiscovered {
		return eventsv1.ArticleCreatedEvent{}, errors.New("unexpected kafka topic")
	}
	var event eventsv1.ArticleDiscoveredEvent
	if err := articleflowkafka.UnmarshalJSON(message.Value, &event); err != nil {
		return eventsv1.ArticleCreatedEvent{}, err
	}
	return handler.ingest.IngestDiscovered(ctx, event)
}


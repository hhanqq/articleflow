package usecase

import (
	"context"
	"io"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type RSSParser func(io.Reader) ([]eventsv1.ArticleDiscoveredEvent, error)

type DiscoveredPublisher struct {
	producer articleflowkafka.Producer
}

func NewDiscoveredPublisher(producer articleflowkafka.Producer) *DiscoveredPublisher {
	return &DiscoveredPublisher{producer: producer}
}

func (publisher *DiscoveredPublisher) PublishFromRSS(ctx context.Context, reader io.Reader, parser RSSParser) error {
	events, err := parser(reader)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := event.Validate(); err != nil {
			return err
		}
		payload, err := articleflowkafka.MarshalJSON(event)
		if err != nil {
			return err
		}
		if err := publisher.producer.Publish(ctx, articleflowkafka.Message{
			Topic: eventsv1.TopicArticleDiscovered,
			Key:   event.ExternalID,
			Value: payload,
		}); err != nil {
			return err
		}
	}
	return nil
}


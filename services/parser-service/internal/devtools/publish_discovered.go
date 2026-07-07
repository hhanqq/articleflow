package devtools

import (
	"context"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

func PublishSampleDiscovered(ctx context.Context, producer articleflowkafka.Producer) error {
	event := eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "dev-habr-article-1",
		URL:          "https://habr.com/ru/articles/dev-articleflow-kafka/",
		Title:        "Articleflow dev Kafka event",
		Summary:      "Development event for checking Kafka to article-service to Postgres flow.",
		Author:       "articleflow-dev",
		Tags:         []string{"go", "kafka", "articleflow"},
		Language:     "ru",
		PublishedAt:  time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Now().UTC(),
	}
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		return err
	}
	return producer.Publish(ctx, articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   event.ExternalID,
		Value: payload,
	})
}

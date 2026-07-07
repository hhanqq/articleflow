package usecase

import (
	"context"
	"strings"
	"testing"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
)

func TestPublishDiscoveredFromRSSPublishesKafkaMessages(t *testing.T) {
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Go microservices</title>
      <link>https://habr.com/ru/articles/123/</link>
      <guid>habr-123</guid>
      <description>Short article summary</description>
      <pubDate>Tue, 07 Jul 2026 10:30:00 +0000</pubDate>
    </item>
  </channel>
</rss>`
	producer := articleflowkafka.NewMemoryProducer()
	publisher := NewDiscoveredPublisher(producer)

	if err := publisher.PublishFromRSS(context.Background(), strings.NewReader(rss), habr.ParseRSS); err != nil {
		t.Fatalf("publish from rss failed: %v", err)
	}

	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 kafka message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicArticleDiscovered {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	if messages[0].Key != "habr-123" {
		t.Fatalf("unexpected key: %s", messages[0].Key)
	}
	var event eventsv1.ArticleDiscoveredEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &event); err != nil {
		t.Fatalf("message should contain article discovered event: %v", err)
	}
	if event.Title != "Go microservices" {
		t.Fatalf("unexpected event title: %s", event.Title)
	}
}


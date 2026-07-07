package devtools

import (
	"context"
	"testing"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

func TestPublishSampleDiscoveredPublishesValidEvent(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()

	if err := PublishSampleDiscovered(context.Background(), producer); err != nil {
		t.Fatalf("publish sample discovered: %v", err)
	}

	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicArticleDiscovered {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	if messages[0].Key != "dev-habr-article-1" {
		t.Fatalf("unexpected key: %s", messages[0].Key)
	}
	var event eventsv1.ArticleDiscoveredEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("expected valid event: %v", err)
	}
	if event.URL != "https://habr.com/ru/articles/dev-articleflow-kafka/" {
		t.Fatalf("unexpected url: %s", event.URL)
	}
}

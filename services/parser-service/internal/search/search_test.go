package search

import (
	"context"
	"errors"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type fakeParser struct {
	sourceName string
	candidates []parserv1.ArticleCandidate
	err        error
}

func (parser fakeParser) SourceName() string {
	return parser.sourceName
}

func (parser fakeParser) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	if parser.err != nil {
		return nil, parser.err
	}
	return parser.candidates, nil
}

func TestSearchAndPublishPublishesCandidates(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{
			sourceName: "habr",
			candidates: []parserv1.ArticleCandidate{
				{
					SourceName:  "habr",
					ExternalID:  "habr-123",
					URL:         "https://habr.com/ru/articles/123/",
					Title:       "Go and Kafka",
					Content:     "Full article content from Habr",
					PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
				},
			},
		},
	})

	candidates, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicArticleDiscovered {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	var event eventsv1.ArticleDiscoveredEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &event); err != nil {
		t.Fatalf("unmarshal discovered event: %v", err)
	}
	if event.Content != "Full article content from Habr" {
		t.Fatalf("unexpected event content: %s", event.Content)
	}
}

func TestSearchAndPublishPublishesFailureEvent(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{sourceName: "habr", err: errors.New("habr unavailable")},
	})

	_, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr"},
		Limit:   10,
	})

	if err == nil {
		t.Fatal("expected parser error")
	}
	messages := producer.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 failure message, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicParserJobFailed {
		t.Fatalf("unexpected topic: %s", messages[0].Topic)
	}
	var event eventsv1.ParserJobFailedEvent
	if err := articleflowkafka.UnmarshalJSON(messages[0].Value, &event); err != nil {
		t.Fatalf("unmarshal failure event: %v", err)
	}
	if event.SourceName != "habr" {
		t.Fatalf("unexpected source name: %s", event.SourceName)
	}
	if event.Query != "go kafka" {
		t.Fatalf("unexpected query: %s", event.Query)
	}
}

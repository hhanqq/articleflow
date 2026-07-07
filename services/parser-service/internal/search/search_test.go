package search

import (
	"context"
	"testing"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type fakeParser struct {
	sourceName string
	candidates []parserv1.ArticleCandidate
}

func (parser fakeParser) SourceName() string {
	return parser.sourceName
}

func (parser fakeParser) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	return parser.candidates, nil
}

func TestSearchAndPublishPublishesCandidates(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{
			sourceName: "habr",
			candidates: []parserv1.ArticleCandidate{
				{
					SourceName: "habr",
					ExternalID: "habr-123",
					URL:        "https://habr.com/ru/articles/123/",
					Title:      "Go and Kafka",
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
}


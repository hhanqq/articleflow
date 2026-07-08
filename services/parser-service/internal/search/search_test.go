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

func TestSearchAndPublishKeepsSuccessfulSourcesWhenAnotherSourceFails(t *testing.T) {
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
				},
			},
		},
		fakeParser{sourceName: "vc", err: errors.New("vc unavailable")},
	})

	candidates, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr", "vc"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("expected partial success, got %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	messages := producer.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected discovered and failure messages, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicArticleDiscovered || messages[1].Topic != eventsv1.TopicParserJobFailed {
		t.Fatalf("unexpected topics: %s, %s", messages[0].Topic, messages[1].Topic)
	}
}

func TestSearchAndPublishNormalizesAndDeduplicatesCandidates(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{
			sourceName: "vc",
			candidates: []parserv1.ArticleCandidate{
				{
					SourceName: " vc ",
					URL:        " https://vc.ru/dev/123-go-kafka?utm_source=rss ",
					Title:      " Go Kafka ",
					Summary:    " First ",
				},
			},
		},
		fakeParser{
			sourceName: "vc_rss",
			candidates: []parserv1.ArticleCandidate{
				{
					SourceName: "vc_rss",
					ExternalID: "rss-duplicate",
					URL:        "https://vc.ru/dev/123-go-kafka",
					Title:      "Go Kafka duplicate",
				},
			},
		},
	})

	candidates, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"vc", "vc_rss"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 deduplicated candidate, got %d", len(candidates))
	}
	if candidates[0].URL != "https://vc.ru/dev/123-go-kafka" {
		t.Fatalf("expected normalized url, got %s", candidates[0].URL)
	}
	if candidates[0].ExternalID == "" {
		t.Fatal("expected stable external id")
	}
	if len(producer.Messages()) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(producer.Messages()))
	}
}

func TestSearchAndPublishBalancesReturnedCandidatesAcrossSources(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{
			sourceName: "habr",
			candidates: []parserv1.ArticleCandidate{
				{SourceName: "habr", ExternalID: "h1", URL: "https://habr.com/1", Title: "Siberia Habr 1"},
				{SourceName: "habr", ExternalID: "h2", URL: "https://habr.com/2", Title: "Siberia Habr 2"},
				{SourceName: "habr", ExternalID: "h3", URL: "https://habr.com/3", Title: "Siberia Habr 3"},
			},
		},
		fakeParser{
			sourceName: "vc",
			candidates: []parserv1.ArticleCandidate{
				{SourceName: "vc", ExternalID: "v1", URL: "https://vc.ru/1", Title: "Siberia VC 1"},
				{SourceName: "vc", ExternalID: "v2", URL: "https://vc.ru/2", Title: "Siberia VC 2"},
				{SourceName: "vc", ExternalID: "v3", URL: "https://vc.ru/3", Title: "Siberia VC 3"},
			},
		},
	})

	candidates, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "Сибирь",
		Sources: []string{"habr", "vc"},
		Limit:   4,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(candidates) != 4 {
		t.Fatalf("expected 4 candidates, got %d", len(candidates))
	}
	sources := []string{candidates[0].SourceName, candidates[1].SourceName, candidates[2].SourceName, candidates[3].SourceName}
	expected := []string{"habr", "vc", "habr", "vc"}
	for index := range expected {
		if sources[index] != expected[index] {
			t.Fatalf("expected balanced sources %#v, got %#v", expected, sources)
		}
	}
	if len(producer.Messages()) != 6 {
		t.Fatalf("expected all discovered candidates to be published, got %d", len(producer.Messages()))
	}
}

func TestSelectedSourcesUsesParserRegistrationOrderWhenQueryOmitsSources(t *testing.T) {
	usecase := NewUsecase(articleflowkafka.NewMemoryProducer(), []Parser{
		fakeParser{sourceName: "habr"},
		fakeParser{sourceName: "vc"},
		fakeParser{sourceName: "dzen"},
	})

	sources := usecase.selectedSources(parserv1.SearchQuery{})

	expected := []string{"habr", "vc", "dzen"}
	for index := range expected {
		if sources[index] != expected[index] {
			t.Fatalf("expected source order %#v, got %#v", expected, sources)
		}
	}
}

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
	strategy   string
	candidates []parserv1.ArticleCandidate
	err        error
}

type failingProducer struct {
	err error
}

func (producer failingProducer) Publish(context.Context, articleflowkafka.Message) error {
	return producer.err
}

func (parser fakeParser) SourceName() string {
	return parser.sourceName
}

func (parser fakeParser) Strategy() string {
	return parser.strategy
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
			strategy:   "html_rss",
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

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Candidates))
	}
	if len(result.SourceStats) != 1 {
		t.Fatalf("expected source stats, got %d", len(result.SourceStats))
	}
	if result.SourceStats[0].FoundCount != 1 || result.SourceStats[0].AcceptedCount != 1 || result.SourceStats[0].ReturnedCount != 1 {
		t.Fatalf("unexpected source stats: %#v", result.SourceStats[0])
	}
	if result.SourceStats[0].Strategy != "html_rss" {
		t.Fatalf("expected source strategy, got %#v", result.SourceStats[0])
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

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr"},
		Limit:   10,
	})

	if err == nil {
		t.Fatal("expected parser error")
	}
	if len(result.SourceStats) != 1 {
		t.Fatalf("expected failed source stats, got %d", len(result.SourceStats))
	}
	if result.SourceStats[0].Status != parserv1.SourceStatusFailed {
		t.Fatalf("expected failed source status, got %s", result.SourceStats[0].Status)
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

func TestSearchAndPublishDoesNotMaskParserErrorWhenFailureEventPublishFails(t *testing.T) {
	parserErr := errors.New("dzen search redirected to auth")
	usecase := NewUsecase(failingProducer{err: errors.New("unknown kafka topic")}, []Parser{
		fakeParser{sourceName: "dzen", err: parserErr},
	})

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "Турция",
		Sources: []string{"dzen"},
		Limit:   10,
	})

	if !errors.Is(err, parserErr) {
		t.Fatalf("expected parser error, got %v", err)
	}
	if len(result.SourceStats) != 1 {
		t.Fatalf("expected source stats, got %d", len(result.SourceStats))
	}
	if result.SourceStats[0].Error != parserErr.Error() {
		t.Fatalf("expected source error in stats, got %#v", result.SourceStats[0])
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

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"habr", "vc"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("expected partial success, got %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Candidates))
	}
	if len(result.SourceStats) != 2 {
		t.Fatalf("expected stats for both sources, got %d", len(result.SourceStats))
	}
	if result.SourceStats[1].Status != parserv1.SourceStatusFailed {
		t.Fatalf("expected second source failed, got %#v", result.SourceStats[1])
	}
	messages := producer.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected discovered and failure messages, got %d", len(messages))
	}
	if messages[0].Topic != eventsv1.TopicParserJobFailed || messages[1].Topic != eventsv1.TopicArticleDiscovered {
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

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "go kafka",
		Sources: []string{"vc", "vc_rss"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 deduplicated candidate, got %d", len(result.Candidates))
	}
	if result.Candidates[0].URL != "https://vc.ru/dev/123-go-kafka" {
		t.Fatalf("expected normalized url, got %s", result.Candidates[0].URL)
	}
	if result.Candidates[0].ExternalID == "" {
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
				{SourceName: "habr", ExternalID: "h1", URL: "https://habr.com/1", Title: "Сибирь Habr 1"},
				{SourceName: "habr", ExternalID: "h2", URL: "https://habr.com/2", Title: "Сибирь Habr 2"},
				{SourceName: "habr", ExternalID: "h3", URL: "https://habr.com/3", Title: "Сибирь Habr 3"},
			},
		},
		fakeParser{
			sourceName: "vc",
			candidates: []parserv1.ArticleCandidate{
				{SourceName: "vc", ExternalID: "v1", URL: "https://vc.ru/1", Title: "Сибирь VC 1"},
				{SourceName: "vc", ExternalID: "v2", URL: "https://vc.ru/2", Title: "Сибирь VC 2"},
				{SourceName: "vc", ExternalID: "v3", URL: "https://vc.ru/3", Title: "Сибирь VC 3"},
			},
		},
	})

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "Сибирь",
		Sources: []string{"habr", "vc"},
		Limit:   4,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(result.Candidates) != 4 {
		t.Fatalf("expected 4 candidates, got %d", len(result.Candidates))
	}
	sources := []string{result.Candidates[0].SourceName, result.Candidates[1].SourceName, result.Candidates[2].SourceName, result.Candidates[3].SourceName}
	expected := []string{"habr", "vc", "habr", "vc"}
	for index := range expected {
		if sources[index] != expected[index] {
			t.Fatalf("expected balanced sources %#v, got %#v", expected, sources)
		}
	}
	if len(producer.Messages()) != 4 {
		t.Fatalf("expected returned candidates to be published, got %d", len(producer.Messages()))
	}
	for _, stat := range result.SourceStats {
		if stat.FoundCount != 3 || stat.AcceptedCount != 3 || stat.ReturnedCount != 2 || stat.PublishedCount != 2 {
			t.Fatalf("unexpected balanced stat: %#v", stat)
		}
	}
}

func TestSearchAndPublishFiltersIrrelevantCandidatesAndSortsByRelevance(t *testing.T) {
	producer := articleflowkafka.NewMemoryProducer()
	usecase := NewUsecase(producer, []Parser{
		fakeParser{
			sourceName: "vc",
			candidates: []parserv1.ArticleCandidate{
				{
					SourceName:  "vc",
					ExternalID:  "low",
					URL:         "https://vc.ru/travel/low",
					Title:       "Путешествие по Японии",
					Summary:     "Короткая заметка",
					PublishedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
				},
				{
					SourceName: "vc",
					ExternalID: "skip",
					URL:        "https://vc.ru/food/skip",
					Title:      "Обзор кофеен",
					Summary:    "Городские места",
				},
				{
					SourceName:  "vc",
					ExternalID:  "high",
					URL:         "https://vc.ru/travel/high",
					Title:       "Путешествие в Японию",
					Summary:     "Путешествие, бюджет и маршрут",
					Tags:        []string{"путешествие"},
					PublishedAt: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC),
				},
			},
		},
	})

	result, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    "путешествие япония",
		Sources: []string{"vc"},
		Limit:   10,
	})

	if err != nil {
		t.Fatalf("search and publish failed: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 relevant candidates, got %d", len(result.Candidates))
	}
	if result.Candidates[0].ExternalID != "high" {
		t.Fatalf("expected highest relevance candidate first, got %s", result.Candidates[0].ExternalID)
	}
	stat := result.SourceStats[0]
	if stat.FoundCount != 3 || stat.AcceptedCount != 2 || stat.FilteredCount != 1 || stat.ReturnedCount != 2 {
		t.Fatalf("unexpected relevance stats: %#v", stat)
	}
	if len(producer.Messages()) != 2 {
		t.Fatalf("expected only relevant candidates to be published, got %d", len(producer.Messages()))
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

package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/config"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

type fakeRuntimeConsumer struct {
	messages []articleflowkafka.Message
	commits  []string
	closed   bool
}

func (consumer *fakeRuntimeConsumer) Fetch(context.Context) (articleflowkafka.Message, error) {
	if len(consumer.messages) == 0 {
		return articleflowkafka.Message{}, io.EOF
	}
	message := consumer.messages[0]
	consumer.messages = consumer.messages[1:]
	return message, nil
}

func (consumer *fakeRuntimeConsumer) Commit(_ context.Context, message articleflowkafka.Message) error {
	consumer.commits = append(consumer.commits, message.Key)
	return nil
}

func (consumer *fakeRuntimeConsumer) Close() error {
	consumer.closed = true
	return nil
}

func TestRunConsumesDiscoveredArticleMessage(t *testing.T) {
	payload, err := articleflowkafka.MarshalJSON(eventsv1.ArticleDiscoveredEvent{
		SourceName:   "habr",
		ExternalID:   "habr-1",
		URL:          "https://habr.com/ru/articles/1/",
		Title:        "Go Kafka",
		DiscoveredAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	consumer := &fakeRuntimeConsumer{
		messages: []articleflowkafka.Message{{
			Topic: eventsv1.TopicArticleDiscovered,
			Key:   "habr-1",
			Value: payload,
		}},
	}
	application := New(config.Config{
		ServiceName:              "article-service",
		GRPCAddr:                 ":0",
		KafkaBrokers:             "localhost:9092",
		ArticleDiscoveredTopic:   eventsv1.TopicArticleDiscovered,
		ArticleConsumerGroupID:   "article-service-test",
		ConsumerMaxMessages:      1,
		StorageDriver:            "memory",
	})
	application.consumerFactory = func(config.Config) runtimeConsumer {
		return consumer
	}

	if err := application.Run(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(consumer.commits) != 1 {
		t.Fatalf("expected 1 committed message, got %d", len(consumer.commits))
	}
	if !consumer.closed {
		t.Fatal("expected consumer to be closed")
	}
}

func TestHTTPHandlerExposesMetricsRoute(t *testing.T) {
	handler := newHTTPHandler("article-service", usecase.NewMemoryArticleStore())
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Body.String() == "" {
		t.Fatal("expected metrics body")
	}
}

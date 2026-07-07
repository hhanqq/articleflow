package eventsv1

import (
	"testing"
	"time"
)

func TestArticleEventTopics(t *testing.T) {
	if TopicArticleDiscovered != "article.discovered.v1" {
		t.Fatalf("unexpected discovered topic: %s", TopicArticleDiscovered)
	}
	if TopicArticleCreated != "article.created.v1" {
		t.Fatalf("unexpected created topic: %s", TopicArticleCreated)
	}
}

func TestArticleDiscoveredEventValidate(t *testing.T) {
	event := ArticleDiscoveredEvent{
		SourceName:  "habr",
		ExternalID:  "habr-123",
		URL:         "https://habr.com/ru/articles/123/",
		Title:       "Go microservices",
		DiscoveredAt: time.Now().UTC(),
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("expected valid event, got error: %v", err)
	}
}

func TestArticleDiscoveredEventValidateRequiresURL(t *testing.T) {
	event := ArticleDiscoveredEvent{
		SourceName:  "habr",
		ExternalID:  "habr-123",
		Title:       "Go microservices",
		DiscoveredAt: time.Now().UTC(),
	}

	if err := event.Validate(); err == nil {
		t.Fatal("expected validation error for empty URL")
	}
}


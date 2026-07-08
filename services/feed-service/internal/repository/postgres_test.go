package repository

import (
	"strings"
	"testing"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

func TestBuildUpsertFeedItemQuery(t *testing.T) {
	event := eventsv1.FeedItemScoredEvent{
		ArticleID:   "habr:1",
		SourceName:  "habr",
		URL:         "https://habr.com/1",
		Title:       "Go Kafka",
		Summary:     "Streaming",
		Tags:        []string{"go", "kafka"},
		Score:       42,
		PublishedAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		ScoredAt:    time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
	}

	query, args := BuildUpsertFeedItemQuery(event)

	if query == "" {
		t.Fatal("expected query")
	}
	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %d", len(args))
	}
	if args[0] != event.ArticleID {
		t.Fatalf("unexpected first arg: %v", args[0])
	}
}

func TestBuildUpsertFeedItemQueryNormalizesNilTags(t *testing.T) {
	event := eventsv1.FeedItemScoredEvent{
		ArticleID:  "habr:1",
		SourceName: "habr",
		URL:        "https://habr.com/1",
		Title:      "Go Kafka",
		ScoredAt:   time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
	}

	_, args := BuildUpsertFeedItemQuery(event)

	tags, ok := args[5].([]string)
	if !ok {
		t.Fatalf("expected tags arg to be []string, got %T", args[5])
	}
	if tags == nil {
		t.Fatal("expected non-nil empty tags slice")
	}
	if len(tags) != 0 {
		t.Fatalf("expected empty tags, got %#v", tags)
	}
}

func TestBuildListFeedItemsQueryNormalizesLimit(t *testing.T) {
	query, args := BuildListFeedItemsQuery(0)

	if query == "" {
		t.Fatal("expected query")
	}
	if !strings.Contains(query, "array_to_json(tags)::text") {
		t.Fatalf("expected tags JSON projection, got %s", query)
	}
	if !strings.Contains(query, "ROW_NUMBER() OVER (PARTITION BY lower(source_name)") {
		t.Fatalf("expected source-balanced row numbers, got %s", query)
	}
	if !strings.Contains(query, "ORDER BY source_rank ASC") {
		t.Fatalf("expected source-balanced ordering, got %s", query)
	}
	if args[0] != 20 {
		t.Fatalf("expected default limit 20, got %v", args[0])
	}
}

func TestDecodeTagsJSON(t *testing.T) {
	tags, err := DecodeTagsJSON(`["go","kafka"]`)
	if err != nil {
		t.Fatalf("decode tags: %v", err)
	}
	if len(tags) != 2 || tags[0] != "go" || tags[1] != "kafka" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

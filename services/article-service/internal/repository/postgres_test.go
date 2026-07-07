package repository

import (
	"strings"
	"testing"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

func TestBuildUpsertArticleQuery(t *testing.T) {
	query, args := BuildUpsertArticleQuery(articlev1.Article{
		ID:          "article-1",
		SourceName:  "habr",
		ExternalID:  "habr-123",
		URL:         "https://habr.com/ru/articles/123/",
		Title:       "Go microservices",
		Summary:     "Summary",
		Content:     "Content",
		Author:      "sergey",
		Tags:        []string{"go", "kafka"},
		Language:    "ru",
		PublishedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
		ParsedAt:    time.Date(2026, 7, 7, 10, 35, 0, 0, time.UTC),
	})

	if !strings.Contains(query, "INSERT INTO articles") {
		t.Fatalf("expected insert query, got %s", query)
	}
	if !strings.Contains(query, "ON CONFLICT (url) DO UPDATE") {
		t.Fatalf("expected upsert query, got %s", query)
	}
	if len(args) != 12 {
		t.Fatalf("expected 12 args, got %d", len(args))
	}
	if args[0] != "article-1" {
		t.Fatalf("unexpected first arg: %v", args[0])
	}
}


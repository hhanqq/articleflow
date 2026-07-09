package dzen

import (
	"strings"
	"testing"
)

func TestParseArticleHTMLUsesJSONLDAndMeta(t *testing.T) {
	article, err := ParseArticleHTML(strings.NewReader(articleHTML("Маршрут по Турции", "Короткое описание", "https://dzen.ru/a/turkey")), "https://dzen.ru/a/turkey")
	if err != nil {
		t.Fatalf("parse article: %v", err)
	}
	if article.SourceName != SourceName {
		t.Fatalf("unexpected source: %s", article.SourceName)
	}
	if article.Title != "Маршрут по Турции" {
		t.Fatalf("unexpected title: %s", article.Title)
	}
	if article.Summary != "Короткое описание" {
		t.Fatalf("unexpected summary: %s", article.Summary)
	}
	if article.ExternalID == "" {
		t.Fatal("expected stable external id")
	}
	if len(article.Tags) != 2 {
		t.Fatalf("expected tags from json-ld, got %#v", article.Tags)
	}
}

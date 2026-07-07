package habr

import (
	"os"
	"testing"
)

func TestParseArticleHTML(t *testing.T) {
	html, err := os.Open("testdata/article.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer html.Close()

	article, err := ParseArticleHTML(html, "https://habr.com/ru/articles/900001/")

	if err != nil {
		t.Fatalf("parse article html failed: %v", err)
	}
	if article.Title != "Go и Kafka в микросервисах" {
		t.Fatalf("unexpected title: %s", article.Title)
	}
	if article.Author != "sergey" {
		t.Fatalf("unexpected author: %s", article.Author)
	}
	if len(article.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(article.Tags))
	}
	if len(article.Content) < 40 {
		t.Fatalf("expected full content, got %q", article.Content)
	}
}


package habr

import (
	"os"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestParseSearchHTML(t *testing.T) {
	html, err := os.Open("testdata/search.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer html.Close()

	candidates, err := ParseSearchHTML(html, parserv1.SearchQuery{Text: "go kafka", Limit: 10})

	if err != nil {
		t.Fatalf("parse search html failed: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].Title != "Go и Kafka в микросервисах" {
		t.Fatalf("unexpected title: %s", candidates[0].Title)
	}
	if candidates[0].URL != "https://habr.com/ru/articles/900001/" {
		t.Fatalf("unexpected url: %s", candidates[0].URL)
	}
	if candidates[0].SourceName != SourceName {
		t.Fatalf("unexpected source: %s", candidates[0].SourceName)
	}
}


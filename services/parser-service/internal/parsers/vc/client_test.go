package vc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestClientSearchEnrichesRSSCandidatesFromArticleHTML(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/rss", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/rss+xml")
		_, _ = response.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
  <item>
    <title>Xiaomi RSS title</title>
    <link>` + server.URL + `/transport/3017516-xiaomi</link>
    <guid>3017516</guid>
    <description>RSS summary only</description>
    <pubDate>Wed, 08 Jul 2026 11:32:12 +0300</pubDate>
  </item>
</channel></rss>`))
	})
	mux.HandleFunc("/transport/3017516-xiaomi", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write([]byte(`<!doctype html><html><head>
<meta name="author" content="Артур Томилко">
<script type="application/ld+json">{
  "@context":"https://schema.org",
  "@type":"NewsArticle",
  "headline":"Xiaomi HTML title",
  "description":"HTML summary",
  "text":"HTML full article text #xiaomi",
  "url":"` + server.URL + `/transport/3017516-xiaomi",
  "datePublished":"2026-07-08T08:32:12.000Z",
  "author":{"name":"Артур Томилко"},
  "keywords":["xiaomi","transport"]
}</script>
</head><body></body></html>`))
	})

	client := NewClient(ClientOptions{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Language:   "ru",
	})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "Xiaomi", Limit: 5})

	if err != nil {
		t.Fatalf("search vc: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	candidate := candidates[0]
	if candidate.Title != "Xiaomi HTML title" {
		t.Fatalf("expected HTML title, got %s", candidate.Title)
	}
	if candidate.Summary != "HTML summary" {
		t.Fatalf("expected HTML summary, got %s", candidate.Summary)
	}
	if candidate.Content != "HTML full article text #xiaomi" {
		t.Fatalf("expected HTML content, got %s", candidate.Content)
	}
	if candidate.Author != "Артур Томилко" {
		t.Fatalf("expected HTML author, got %s", candidate.Author)
	}
	if len(candidate.Tags) != 2 || candidate.Tags[0] != "xiaomi" || candidate.Tags[1] != "transport" {
		t.Fatalf("unexpected tags: %#v", candidate.Tags)
	}
}

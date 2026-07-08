package vc

import (
	"strings"
	"testing"
)

func TestParseArticleHTMLExtractsStructuredArticleData(t *testing.T) {
	html := `<!doctype html>
<html>
<head>
  <meta property="og:title" content="Xiaomi анонсировала Sky Nomad — Транспорт на vc.ru">
  <meta name="description" content="Короткое описание из meta">
  <meta name="author" content="Артур Томилко">
  <link rel="canonical" href="https://vc.ru/transport/3017516-xiaomi-predstavila-subbrend-sky-nomad">
  <script type="application/ld+json">{
    "@context":"https://schema.org",
    "@graph":[{
      "@type":"NewsArticle",
      "headline":"Xiaomi анонсировала автомобильный суббренд Sky Nomad",
      "description":"Описание из JSON-LD",
      "text":"Под новой маркой компания планирует выпускать машины. #новости #xiaomi",
      "url":"https://vc.ru/transport/3017516-xiaomi-predstavila-subbrend-sky-nomad",
      "datePublished":"2026-07-08T08:32:12.000Z",
      "author":{"@type":"Person","name":"Артур Томилко"},
      "keywords":["новости","xiaomi"]
    }]
  }</script>
</head>
<body></body>
</html>`

	article, err := ParseArticleHTML(strings.NewReader(html), "https://vc.ru/transport/3017516-xiaomi-predstavila-subbrend-sky-nomad?from=rss")

	if err != nil {
		t.Fatalf("parse vc article html: %v", err)
	}
	if article.Title != "Xiaomi анонсировала автомобильный суббренд Sky Nomad" {
		t.Fatalf("unexpected title: %s", article.Title)
	}
	if article.Content != "Под новой маркой компания планирует выпускать машины. #новости #xiaomi" {
		t.Fatalf("unexpected content: %s", article.Content)
	}
	if article.Summary != "Описание из JSON-LD" {
		t.Fatalf("unexpected summary: %s", article.Summary)
	}
	if article.Author != "Артур Томилко" {
		t.Fatalf("unexpected author: %s", article.Author)
	}
	if article.URL != "https://vc.ru/transport/3017516-xiaomi-predstavila-subbrend-sky-nomad" {
		t.Fatalf("unexpected url: %s", article.URL)
	}
	if len(article.Tags) != 2 || article.Tags[0] != "новости" || article.Tags[1] != "xiaomi" {
		t.Fatalf("unexpected tags: %#v", article.Tags)
	}
	if article.PublishedAt.IsZero() {
		t.Fatal("expected published time from JSON-LD")
	}
}

func TestParseArticleHTMLPrefersInitialStateBlocksForFullContent(t *testing.T) {
	html := `<!doctype html>
<html>
<head>
  <meta name="description" content="Короткое описание">
</head>
<body>
<script>
window.__INITIAL_STATE__ = {
  "entry@3017516": {
    "title": "Xiaomi HTML state title",
    "date": 1783499532,
    "url": "https://vc.ru/transport/3017516-xiaomi",
    "author": {"name": "Артур Томилко"},
    "keywords": ["новости", "xiaomi"],
    "blocks": [
      {"type": "text", "data": {"text": "\u003Cp\u003EПервый абзац из state.\u003C/p\u003E"}},
      {"type": "list", "data": {"items": [
        "Первый пункт \u003Ca href=\"https://example.com\"\u003Eсо ссылкой\u003C/a\u003E.",
        "Второй пункт."
      ]}},
      {"type": "text", "data": {"text": "\u003Cp\u003E#новости #xiaomi\u003C/p\u003E"}}
    ]
  }
};
</script>
</body>
</html>`

	article, err := ParseArticleHTML(strings.NewReader(html), "https://vc.ru/transport/3017516-xiaomi?from=rss")

	if err != nil {
		t.Fatalf("parse vc article html: %v", err)
	}
	if article.Title != "Xiaomi HTML state title" {
		t.Fatalf("unexpected title: %s", article.Title)
	}
	expectedContent := "Первый абзац из state. Первый пункт со ссылкой. Второй пункт. #новости #xiaomi"
	if article.Content != expectedContent {
		t.Fatalf("unexpected content: %s", article.Content)
	}
	if article.Author != "Артур Томилко" {
		t.Fatalf("unexpected author: %s", article.Author)
	}
	if len(article.Tags) != 2 || article.Tags[0] != "новости" || article.Tags[1] != "xiaomi" {
		t.Fatalf("unexpected tags: %#v", article.Tags)
	}
	if article.PublishedAt.IsZero() {
		t.Fatal("expected published time from state date")
	}
}

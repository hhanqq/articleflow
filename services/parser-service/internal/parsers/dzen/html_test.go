package dzen

import (
	"strings"
	"testing"
	"time"
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

func TestParseArticleHTMLUsesEmbeddedDzenPublishTime(t *testing.T) {
	article, err := ParseArticleHTML(strings.NewReader(`<!doctype html>
<html>
<head>
	<link rel="canonical" href="https://dzen.ru/a/live">
	<meta property="og:title" content="Новости">
	<meta name="description" content="Описание">
	<script>window._params={"publishTime":1783603643721,"publishDate":"2026-07-09"}</script>
</head>
<body><article><p>Текст новости.</p></article></body>
</html>`), "https://dzen.ru/a/live")
	if err != nil {
		t.Fatalf("parse article: %v", err)
	}
	expected := time.UnixMilli(1783603643721).UTC()
	if !article.PublishedAt.Equal(expected) {
		t.Fatalf("unexpected published_at: got %s want %s", article.PublishedAt, expected)
	}
}

package habr

import (
	"strings"
	"testing"
)

func TestParseRSSProducesDiscoveredEvents(t *testing.T) {
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Habr</title>
    <item>
      <title>Go microservices</title>
      <link>https://habr.com/ru/articles/123/</link>
      <guid>https://habr.com/ru/articles/123/</guid>
      <description>Short article summary</description>
      <author>author@example.com</author>
      <category>go</category>
      <category>microservices</category>
      <pubDate>Tue, 07 Jul 2026 10:30:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

	events, err := ParseRSS(strings.NewReader(rss))

	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	event := events[0]
	if event.SourceName != SourceName {
		t.Fatalf("unexpected source name: %s", event.SourceName)
	}
	if event.Title != "Go microservices" {
		t.Fatalf("unexpected title: %s", event.Title)
	}
	if event.URL != "https://habr.com/ru/articles/123/" {
		t.Fatalf("unexpected url: %s", event.URL)
	}
	if len(event.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(event.Tags))
	}
	if event.PublishedAt.IsZero() {
		t.Fatal("expected published at to be parsed")
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("event should be valid: %v", err)
	}
}


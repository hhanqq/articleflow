package rssfeed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestClientSearchFiltersRSSItemsByQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/rss" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/rss+xml")
		_, _ = response.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Kafka для продуктовой аналитики</title>
      <link>https://vc.ru/dev/1001-kafka</link>
      <guid>vc-1001</guid>
      <description>Как команды используют Kafka и Go.</description>
      <category>dev</category>
      <category>kafka</category>
      <pubDate>Tue, 07 Jul 2026 10:30:00 +0000</pubDate>
    </item>
    <item>
      <title>Маркетинг без разработки</title>
      <link>https://vc.ru/marketing/1002</link>
      <guid>vc-1002</guid>
      <description>Не про инженерные темы.</description>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()
	client := NewClient(ClientOptions{
		SourceName: "vc",
		FeedURL:    server.URL + "/rss",
		HTTPClient: server.Client(),
	})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "go kafka", Limit: 10})

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].SourceName != "vc" {
		t.Fatalf("unexpected source: %s", candidates[0].SourceName)
	}
	if candidates[0].ExternalID != "vc-1001" {
		t.Fatalf("unexpected external id: %s", candidates[0].ExternalID)
	}
	if candidates[0].Content == "" {
		t.Fatal("expected rss description as initial content")
	}
}

func TestClientSearchIgnoresStopWordsWhenFiltering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/rss+xml")
		_, _ = response.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Источник РБК рассказал о планах компании</title>
      <link>https://vc.ru/news/2001</link>
      <guid>vc-2001</guid>
      <description>В компании говорят о реструктуризации.</description>
    </item>
    <item>
      <title>Маршрут путешествия в Китай для предпринимателей</title>
      <link>https://vc.ru/travel/2002</link>
      <guid>vc-2002</guid>
      <description>Пекин, Шанхай и деловые поездки.</description>
    </item>
  </channel>
</rss>`))
	}))
	defer server.Close()
	client := NewClient(ClientOptions{
		SourceName: "vc",
		FeedURL:    server.URL,
		HTTPClient: server.Client(),
	})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "путешествие в китай", Limit: 10})

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ExternalID != "vc-2002" {
		t.Fatalf("unexpected candidate: %s", candidates[0].ExternalID)
	}
}

func TestClientSearchParsesAtomFeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/atom+xml")
		_, _ = response.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Путешествие в Японию для предпринимателей</title>
    <link href="https://example.com/japan"/>
    <id>atom-japan-1</id>
    <summary>Маршрут, расходы и деловые встречи.</summary>
    <author><name>Редакция</name></author>
    <category term="travel"/>
    <updated>2026-07-08T08:32:12Z</updated>
  </entry>
</feed>`))
	}))
	defer server.Close()
	client := NewClient(ClientOptions{
		SourceName: "atom_source",
		FeedURL:    server.URL,
		HTTPClient: server.Client(),
	})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "японию", Limit: 10})

	if err != nil {
		t.Fatalf("search atom failed: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 atom candidate, got %d", len(candidates))
	}
	if candidates[0].ExternalID != "atom-japan-1" {
		t.Fatalf("unexpected external id: %s", candidates[0].ExternalID)
	}
	if candidates[0].URL != "https://example.com/japan" {
		t.Fatalf("unexpected url: %s", candidates[0].URL)
	}
	if candidates[0].Author != "Редакция" {
		t.Fatalf("unexpected author: %s", candidates[0].Author)
	}
	if candidates[0].PublishedAt.IsZero() {
		t.Fatal("expected atom updated time")
	}
}

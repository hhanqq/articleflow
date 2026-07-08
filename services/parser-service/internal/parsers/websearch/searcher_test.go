package websearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleSearcherSearchURLsUsesSiteRestrictedQuery(t *testing.T) {
	var rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		rawQuery = request.URL.Query().Get("q")
		if request.URL.Query().Get("key") != "test-key" {
			t.Fatalf("missing api key: %s", request.URL.RawQuery)
		}
		if request.URL.Query().Get("cx") != "test-cx" {
			t.Fatalf("missing cx: %s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"items": [
				{"link": "https://vc.ru/travel/42-china"},
				{"link": "https://example.com/not-vc"},
				{"link": "https://vc.ru/marketing/43"}
			]
		}`))
	}))
	defer server.Close()

	searcher := NewGoogleSearcher(GoogleOptions{
		Endpoint: server.URL,
		APIKey:   "test-key",
		CX:       "test-cx",
		Site:     "vc.ru",
		Client:   server.Client(),
	})

	urls, err := searcher.SearchURLs(context.Background(), "путешествие в китай", 5)

	if err != nil {
		t.Fatalf("search urls: %v", err)
	}
	if rawQuery != "site:vc.ru путешествие в китай" {
		t.Fatalf("unexpected query: %s", rawQuery)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 vc urls, got %#v", urls)
	}
	if urls[0] != "https://vc.ru/travel/42-china" || urls[1] != "https://vc.ru/marketing/43" {
		t.Fatalf("unexpected urls: %#v", urls)
	}
}

func TestBingSearcherSearchURLsUsesSubscriptionHeader(t *testing.T) {
	var apiKey string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		apiKey = request.Header.Get("Ocp-Apim-Subscription-Key")
		if !strings.Contains(request.URL.Query().Get("q"), "site:vc.ru") {
			t.Fatalf("expected site restricted query, got %s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"webPages": {
				"value": [
					{"url": "https://vc.ru/life/100"},
					{"url": "https://not-vc.ru/life/100"}
				]
			}
		}`))
	}))
	defer server.Close()

	searcher := NewBingSearcher(BingOptions{
		Endpoint: server.URL,
		APIKey:   "bing-key",
		Site:     "vc.ru",
		Client:   server.Client(),
	})

	urls, err := searcher.SearchURLs(context.Background(), "китай", 3)

	if err != nil {
		t.Fatalf("search urls: %v", err)
	}
	if apiKey != "bing-key" {
		t.Fatalf("unexpected api key header: %s", apiKey)
	}
	if len(urls) != 1 || urls[0] != "https://vc.ru/life/100" {
		t.Fatalf("unexpected urls: %#v", urls)
	}
}

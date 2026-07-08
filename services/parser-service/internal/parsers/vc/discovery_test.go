package vc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverySearcherSearchURLsReadsSearchPostsAPI(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/v2.10/search/posts", func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("q") != "поездка в японию" {
			t.Fatalf("unexpected query: %s", request.URL.RawQuery)
		}
		if request.URL.Query().Get("markdown") != "false" {
			t.Fatalf("expected markdown=false, got %s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"message": "",
			"result": map[string]any{
				"items": []map[string]any{
					{"type": "entry", "data": map[string]any{"url": "https://vc.ru/travel/1799146-japan?utm_source=feed"}},
					{"type": "entry", "data": map[string]any{"url": "https://vc.ru/life/2179071-korea"}},
					{"type": "entry", "data": map[string]any{"url": "https://vc.ru/life/2179071-korea#comments"}},
				},
			},
		})
	})

	searcher := NewDiscoverySearcher(DiscoverySearcherOptions{
		Endpoint: server.URL + "/v2.10/search/posts",
		Client:   server.Client(),
	})

	urls, err := searcher.SearchURLs(context.Background(), "поездка в японию", 10)

	if err != nil {
		t.Fatalf("search discovery: %v", err)
	}
	expected := []string{
		"https://vc.ru/travel/1799146-japan",
		"https://vc.ru/life/2179071-korea",
	}
	if len(urls) != len(expected) {
		t.Fatalf("expected %d urls, got %#v", len(expected), urls)
	}
	for index := range expected {
		if urls[index] != expected[index] {
			t.Fatalf("unexpected url at %d: %s", index, urls[index])
		}
	}
}

func TestDiscoverySearcherSearchURLSLimitsResults(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/v2.10/search/posts", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"result": {
				"items": [
					{"type": "entry", "data": {"url": "https://vc.ru/a/1"}},
					{"type": "entry", "data": {"url": "https://vc.ru/a/2"}}
				]
			}
		}`))
	})

	searcher := NewDiscoverySearcher(DiscoverySearcherOptions{
		Endpoint: server.URL + "/v2.10/search/posts",
		Client:   server.Client(),
	})

	urls, err := searcher.SearchURLs(context.Background(), "go kafka", 1)

	if err != nil {
		t.Fatalf("search discovery: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://vc.ru/a/1" {
		t.Fatalf("expected first url only, got %#v", urls)
	}
}

func TestDiscoverySearcherSearchURLsPaginatesUntilLimit(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	requests := 0
	mux.HandleFunc("/v2.10/search/posts", func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if request.URL.Query().Get("lastId") != "" {
				t.Fatalf("first request must not include lastId: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{
				"result": {
					"lastId": 20,
					"lastSortingValue": 2000,
					"items": [
						{"type": "entry", "data": {"url": "https://vc.ru/a/1"}},
						{"type": "entry", "data": {"url": "https://vc.ru/a/2"}}
					]
				}
			}`))
		case 2:
			if request.URL.Query().Get("lastId") != "20" {
				t.Fatalf("expected second request lastId=20, got %s", request.URL.RawQuery)
			}
			if request.URL.Query().Get("lastSortingValue") != "2000" {
				t.Fatalf("expected second request lastSortingValue=2000, got %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{
				"result": {
					"lastId": 40,
					"lastSortingValue": 1000,
					"items": [
						{"type": "entry", "data": {"url": "https://vc.ru/a/3"}},
						{"type": "entry", "data": {"url": "https://vc.ru/a/4"}}
					]
				}
			}`))
		default:
			t.Fatalf("unexpected extra request %d", requests)
		}
	})

	searcher := NewDiscoverySearcher(DiscoverySearcherOptions{
		Endpoint: server.URL + "/v2.10/search/posts",
		Client:   server.Client(),
	})

	urls, err := searcher.SearchURLs(context.Background(), "go kafka", 3)

	if err != nil {
		t.Fatalf("search discovery: %v", err)
	}
	expected := []string{"https://vc.ru/a/1", "https://vc.ru/a/2", "https://vc.ru/a/3"}
	if fmt.Sprint(urls) != fmt.Sprint(expected) {
		t.Fatalf("expected paginated urls %#v, got %#v", expected, urls)
	}
	if requests != 2 {
		t.Fatalf("expected 2 page requests, got %d", requests)
	}
}

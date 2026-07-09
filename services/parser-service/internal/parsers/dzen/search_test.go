package dzen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestSearchHTMLExtractsArticleLinksAndParsesArticles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/search":
			if request.URL.Query().Get("query") != "Турция" {
				t.Fatalf("unexpected query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`
				<html><body>
					<a href="/a/first">Первый материал</a>
					<a href="https://dzen.ru/media/travel/second?utm_source=x">Второй материал</a>
					<a href="/profile/editor">Канал</a>
					<a href="/a/first">Дубль</a>
				</body></html>
			`))
		case "/a/first":
			_, _ = response.Write([]byte(articleHTML("Первая Турция", "Описание первого", "https://dzen.ru/a/first")))
		case "/media/travel/second":
			_, _ = response.Write([]byte(articleHTML("Вторая Турция", "Описание второго", "https://dzen.ru/media/travel/second")))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, PublicBaseURL: "https://dzen.ru", HTTPClient: server.Client()})

	candidates, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "Турция", Limit: 5}.Normalize())
	if err != nil {
		t.Fatalf("search dzen: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %#v", len(candidates), candidates)
	}
	if candidates[0].SourceName != SourceName || candidates[0].Title != "Первая Турция" {
		t.Fatalf("unexpected first candidate: %#v", candidates[0])
	}
	if candidates[1].URL != "https://dzen.ru/media/travel/second" {
		t.Fatalf("unexpected second URL: %s", candidates[1].URL)
	}
}

func TestSearchReturnsAuthRedirectError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, "https://sso.passport.yandex.ru/push?retpath=https://dzen.ru/search", http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, PublicBaseURL: "https://dzen.ru", HTTPClient: server.Client()})

	_, err := client.Search(context.Background(), parserv1.SearchQuery{Text: "Турция", Limit: 5}.Normalize())
	if err == nil {
		t.Fatal("expected auth redirect error")
	}
}

func articleHTML(title, description, canonical string) string {
	return `<!doctype html>
<html>
<head>
	<link rel="canonical" href="` + canonical + `">
	<meta property="og:title" content="` + title + `">
	<meta name="description" content="` + description + `">
	<script type="application/ld+json">{
		"@type":"Article",
		"headline":"` + title + `",
		"description":"` + description + `",
		"author":{"name":"Редактор"},
		"datePublished":"2026-07-07T10:00:00Z",
		"keywords":["travel","turkey"]
	}</script>
</head>
<body><article><p>Полный текст про Турцию и поездку.</p></article></body>
</html>`
}

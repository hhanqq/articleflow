package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeParserSourceClient struct {
	sources []parserv1.ParserSource
}

func (client fakeParserSourceClient) ListSources(_ context.Context) ([]parserv1.ParserSource, error) {
	return client.sources, nil
}

func TestSearchSourcesHandlerListsParserSources(t *testing.T) {
	handler := NewSearchSourcesHandler(fakeParserSourceClient{
		sources: []parserv1.ParserSource{
			{Name: "habr", DisplayName: "Habr", Kind: "html_rss", Enabled: true, Searchable: true},
			{Name: "vc", DisplayName: "vc.ru", Kind: "html", Enabled: true, Searchable: true},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/search/sources", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload SearchSourcesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(payload.Sources))
	}
	if payload.Sources[1].Name != "vc" {
		t.Fatalf("unexpected source: %#v", payload.Sources[1])
	}
}

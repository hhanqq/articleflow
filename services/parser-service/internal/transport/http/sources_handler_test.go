package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

func TestSourcesHandlerListsParserSources(t *testing.T) {
	handler := NewSourcesHandler([]parserv1.ParserSource{
		{Name: "habr", DisplayName: "Habr", Kind: "html_rss", Enabled: true, Searchable: true},
		{Name: "vc_rss", DisplayName: "vc.ru RSS", Kind: "rss", Enabled: false, Searchable: true},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/parser/sources", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload SourcesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(payload.Sources))
	}
	if payload.Sources[0].Name != "habr" || !payload.Sources[0].Enabled {
		t.Fatalf("unexpected first source: %#v", payload.Sources[0])
	}
}

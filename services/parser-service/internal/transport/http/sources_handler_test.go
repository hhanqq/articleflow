package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"github.com/hanq/articleflow/services/parser-service/internal/sources"
)

type fakeSourceMetrics struct {
	updates []sourceMetricsUpdate
}

type sourceMetricsUpdate struct {
	name    string
	enabled bool
}

func (metrics *fakeSourceMetrics) ObserveSourceEnabled(name string, enabled bool) {
	metrics.updates = append(metrics.updates, sourceMetricsUpdate{name: name, enabled: enabled})
}

func TestSourcesHandlerListsParserSources(t *testing.T) {
	handler := NewSourcesHandler(sources.NewRuntimeRegistry([]parserv1.ParserSource{
		{Name: "habr", DisplayName: "Habr", Kind: "html_rss", Enabled: true, Searchable: true},
		{Name: "vc_rss", DisplayName: "vc.ru RSS", Kind: "rss", Enabled: false, Searchable: true},
	}))
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

func TestSourcesHandlerUpdatesSourceEnabledState(t *testing.T) {
	registry := sources.NewRuntimeRegistry([]parserv1.ParserSource{
		{Name: "vc", DisplayName: "vc.ru", Kind: "html", Enabled: true, Searchable: true},
	})
	metrics := &fakeSourceMetrics{}
	handler := NewSourcesHandlerWithMetrics(registry, metrics)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/parser/sources/vc", bytes.NewBufferString(`{"enabled":false}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload SourceResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Source.Name != "vc" || payload.Source.Enabled {
		t.Fatalf("unexpected source response: %#v", payload.Source)
	}
	if registry.Enabled("vc") {
		t.Fatal("expected registry source disabled")
	}
	if len(metrics.updates) != 1 || metrics.updates[0].name != "vc" || metrics.updates[0].enabled {
		t.Fatalf("expected source metrics update for disabled vc, got %#v", metrics.updates)
	}
}

package sources

import (
	"testing"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/vc"
)

func TestBuildParsersRegistersBuiltInAndCustomSources(t *testing.T) {
	parsers := BuildParsers(config.Config{
		HabrBaseURL:      "https://habr.com",
		VCRSSFeedURL:     "https://vc.ru/rss",
		VCBaseURL:        "https://vc.ru",
		DzenBaseURL:      "https://dzen.ru",
		DzenRSSFeedURL:   "https://dzen.ru/rss",
		CustomRSSSources: "yandex=https://news.yandex.ru/index.rss",
	})

	names := make(map[string]bool, len(parsers))
	for _, parser := range parsers {
		names[parser.SourceName()] = true
	}

	for _, name := range []string{"habr", "vc", "vc_rss", "dzen", "dzen_rss", "yandex"} {
		if !names[name] {
			t.Fatalf("expected parser %q in registry, got %#v", name, names)
		}
	}
}

func TestBuildRegistryListsSourcesAndExcludesDisabledParsers(t *testing.T) {
	registry := BuildRegistry(config.Config{
		HabrBaseURL:        "https://habr.com",
		VCRSSFeedURL:       "https://vc.ru/rss",
		VCBaseURL:          "https://vc.ru",
		DzenBaseURL:        "https://dzen.ru",
		DzenRSSFeedURL:     "https://dzen.ru/rss",
		DisabledSources:    "vc_rss",
		VCSearchProvider:   "discovery",
		HabrMaxAttempts:    1,
		HabrRetryDelayMS:   0,
		HabrRequestDelayMS: 0,
	})

	if len(registry.Parsers) != 4 {
		t.Fatalf("expected disabled vc_rss parser to be excluded, got %d parsers", len(registry.Parsers))
	}
	byName := make(map[string]SourceInfo, len(registry.Sources))
	for _, source := range registry.Sources {
		byName[source.Name] = source
	}
	if !byName["habr"].Enabled || byName["habr"].Kind != "html_rss" {
		t.Fatalf("unexpected habr source info: %#v", byName["habr"])
	}
	if byName["vc_rss"].Enabled {
		t.Fatalf("expected vc_rss disabled, got %#v", byName["vc_rss"])
	}
	if byName["dzen"].Kind != "html" || !byName["dzen"].Enabled {
		t.Fatalf("unexpected dzen source info: %#v", byName["dzen"])
	}
	if byName["dzen_rss"].Kind != "rss" || !byName["dzen_rss"].Enabled {
		t.Fatalf("unexpected dzen rss source info: %#v", byName["dzen_rss"])
	}
}

func TestBuildParsersRegistersVCWithGoogleSearchProvider(t *testing.T) {
	parsers := BuildParsers(config.Config{
		HabrBaseURL:        "https://habr.com",
		VCBaseURL:          "https://vc.ru",
		VCSearchProvider:   "google",
		GoogleSearchAPIKey: "key",
		GoogleSearchCX:     "cx",
	})

	names := make(map[string]bool, len(parsers))
	for _, parser := range parsers {
		names[parser.SourceName()] = true
	}

	if !names["vc"] {
		t.Fatalf("expected vc parser in registry, got %#v", names)
	}
}

func TestBuildVCURLSearcherDefaultsToDiscoveryProvider(t *testing.T) {
	searcher := buildVCURLSearcher(config.Config{VCBaseURL: "https://vc.ru"})

	if _, ok := searcher.(*vc.DiscoverySearcher); !ok {
		t.Fatalf("expected default discovery searcher, got %T", searcher)
	}
}

func TestBuildVCURLSearcherSupportsExplicitDiscoveryProvider(t *testing.T) {
	searcher := buildVCURLSearcher(config.Config{
		VCBaseURL:        "https://vc.ru",
		VCSearchProvider: "discovery",
	})

	if _, ok := searcher.(*vc.DiscoverySearcher); !ok {
		t.Fatalf("expected discovery searcher, got %T", searcher)
	}
}

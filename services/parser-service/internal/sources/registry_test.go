package sources

import (
	"testing"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
)

func TestBuildParsersRegistersBuiltInAndCustomSources(t *testing.T) {
	parsers := BuildParsers(config.Config{
		HabrBaseURL:      "https://habr.com",
		VCRSSFeedURL:     "https://vc.ru/rss",
		VCBaseURL:        "https://vc.ru",
		CustomRSSSources: "dzen=https://dzen.ru/rss,yandex=https://news.yandex.ru/index.rss",
	})

	names := make(map[string]bool, len(parsers))
	for _, parser := range parsers {
		names[parser.SourceName()] = true
	}

	for _, name := range []string{"habr", "vc", "vc_rss", "dzen", "yandex"} {
		if !names[name] {
			t.Fatalf("expected parser %q in registry, got %#v", name, names)
		}
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

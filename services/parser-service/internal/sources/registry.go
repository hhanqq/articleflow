package sources

import (
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"time"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/rssfeed"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/vc"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/websearch"
	"github.com/hanq/articleflow/services/parser-service/internal/search"
)

type SourceInfo = parserv1.ParserSource

type Registry struct {
	Parsers []search.Parser
	Sources []SourceInfo
}

func BuildParsers(cfg config.Config) []search.Parser {
	return BuildRegistry(cfg).Parsers
}

func BuildRegistry(cfg config.Config) Registry {
	disabled := cfg.DisabledSourceSet()
	registry := Registry{}
	add := func(info SourceInfo, parser search.Parser) {
		info.Enabled = !disabled[info.Name]
		info.Searchable = true
		registry.Sources = append(registry.Sources, info)
		if info.Enabled && parser != nil {
			registry.Parsers = append(registry.Parsers, parser)
		}
	}

	add(SourceInfo{Name: "habr", DisplayName: "Habr", Kind: "html_rss"}, habr.NewClient(habr.ClientOptions{
		BaseURL:     cfg.HabrBaseURL,
		MaxAttempts: cfg.HabrMaxAttempts,
		RetryDelay:  time.Duration(cfg.HabrRetryDelayMS) * time.Millisecond,
		Waiter:      habr.FixedDelayWaiter{Delay: time.Duration(cfg.HabrRequestDelayMS) * time.Millisecond},
	}))
	add(SourceInfo{Name: "vc", DisplayName: "vc.ru", Kind: "html"}, vc.NewClient(vc.ClientOptions{
		BaseURL:     cfg.VCBaseURL,
		Language:    "ru",
		URLSearcher: buildVCURLSearcher(cfg),
	}))
	if cfg.VCRSSFeedURL != "" {
		add(SourceInfo{Name: "vc_rss", DisplayName: "vc.ru RSS", Kind: "rss"}, rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: "vc_rss",
			FeedURL:    cfg.VCRSSFeedURL,
			Language:   "ru",
		}))
	}
	for _, source := range cfg.RSSSourceList() {
		add(SourceInfo{Name: source.Name, DisplayName: source.Name, Kind: "rss"}, rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: source.Name,
			FeedURL:    source.URL,
			Language:   "ru",
		}))
	}
	return registry
}

func buildVCURLSearcher(cfg config.Config) vc.URLSearcher {
	switch cfg.VCSearchProvider {
	case "", "discovery", "vc_discovery":
		return vc.NewDiscoverySearcher(vc.DiscoverySearcherOptions{})
	case "google", "google_cse":
		if cfg.GoogleSearchAPIKey == "" || cfg.GoogleSearchCX == "" {
			return nil
		}
		return websearch.NewGoogleSearcher(websearch.GoogleOptions{
			APIKey: cfg.GoogleSearchAPIKey,
			CX:     cfg.GoogleSearchCX,
			Site:   "vc.ru",
		})
	case "bing":
		if cfg.BingSearchAPIKey == "" {
			return nil
		}
		return websearch.NewBingSearcher(websearch.BingOptions{
			APIKey: cfg.BingSearchAPIKey,
			Site:   "vc.ru",
		})
	default:
		return nil
	}
}

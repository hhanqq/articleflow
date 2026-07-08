package sources

import (
	"time"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/rssfeed"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/vc"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/websearch"
	"github.com/hanq/articleflow/services/parser-service/internal/search"
)

func BuildParsers(cfg config.Config) []search.Parser {
	parsers := []search.Parser{
		habr.NewClient(habr.ClientOptions{
			BaseURL:     cfg.HabrBaseURL,
			MaxAttempts: cfg.HabrMaxAttempts,
			RetryDelay:  time.Duration(cfg.HabrRetryDelayMS) * time.Millisecond,
			Waiter:      habr.FixedDelayWaiter{Delay: time.Duration(cfg.HabrRequestDelayMS) * time.Millisecond},
		}),
		vc.NewClient(vc.ClientOptions{
			BaseURL:     cfg.VCBaseURL,
			Language:    "ru",
			URLSearcher: buildVCURLSearcher(cfg),
		}),
	}
	if cfg.VCRSSFeedURL != "" {
		parsers = append(parsers, rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: "vc_rss",
			FeedURL:    cfg.VCRSSFeedURL,
			Language:   "ru",
		}))
	}
	for _, source := range cfg.RSSSourceList() {
		parsers = append(parsers, rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: source.Name,
			FeedURL:    source.URL,
			Language:   "ru",
		}))
	}
	return parsers
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

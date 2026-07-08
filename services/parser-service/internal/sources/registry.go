package sources

import (
	"time"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/rssfeed"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/vc"
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
			BaseURL:  cfg.VCBaseURL,
			Language: "ru",
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

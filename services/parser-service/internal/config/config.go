package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName        string
	KafkaBrokers       string
	HTTPAddr           string
	HabrBaseURL        string
	HabrMaxAttempts    int
	HabrRetryDelayMS   int
	HabrRequestDelayMS int
	VCBaseURL          string
	VCRSSFeedURL       string
	VCSearchProvider   string
	GoogleSearchAPIKey string
	GoogleSearchCX     string
	BingSearchAPIKey   string
	CustomRSSSources   string
}

func Load() Config {
	return Config{
		ServiceName:        "parser-service",
		KafkaBrokers:       sharedconfig.String("KAFKA_BROKERS", "127.0.0.1:9092"),
		HTTPAddr:           sharedconfig.String("PARSER_HTTP_ADDR", ":8081"),
		HabrBaseURL:        sharedconfig.String("HABR_BASE_URL", "https://habr.com"),
		HabrMaxAttempts:    sharedconfig.Int("HABR_MAX_ATTEMPTS", 3),
		HabrRetryDelayMS:   sharedconfig.Int("HABR_RETRY_DELAY_MS", 500),
		HabrRequestDelayMS: sharedconfig.Int("HABR_REQUEST_DELAY_MS", 500),
		VCBaseURL:          sharedconfig.String("VC_BASE_URL", "https://vc.ru"),
		VCRSSFeedURL:       sharedconfig.String("VC_RSS_FEED_URL", "https://vc.ru/rss"),
		VCSearchProvider:   sharedconfig.String("VC_SEARCH_PROVIDER", "discovery"),
		GoogleSearchAPIKey: sharedconfig.String("GOOGLE_SEARCH_API_KEY", ""),
		GoogleSearchCX:     sharedconfig.String("GOOGLE_SEARCH_CX", ""),
		BingSearchAPIKey:   sharedconfig.String("BING_SEARCH_API_KEY", ""),
		CustomRSSSources:   sharedconfig.String("CUSTOM_RSS_SOURCES", ""),
	}
}

type RSSSource struct {
	Name string
	URL  string
}

func (cfg Config) BrokerList() []string {
	parts := strings.Split(cfg.KafkaBrokers, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}

func (cfg Config) RSSSourceList() []RSSSource {
	parts := strings.Split(cfg.CustomRSSSources, ",")
	sources := make([]RSSSource, 0, len(parts))
	for _, part := range parts {
		name, sourceURL, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		sourceURL = strings.TrimSpace(sourceURL)
		if name == "" || sourceURL == "" {
			continue
		}
		sources = append(sources, RSSSource{Name: name, URL: sourceURL})
	}
	return sources
}

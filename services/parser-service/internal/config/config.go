package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName        string
	KafkaBrokers       string
	HTTPAddr           string
	StorageDriver      string
	PostgresDSN        string
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
	DzenRSSFeedURL     string
	YandexRSSFeedURL   string
	CustomRSSSources   string
	DisabledSources    string
}

func Load() Config {
	return Config{
		ServiceName:        "parser-service",
		KafkaBrokers:       sharedconfig.String("KAFKA_BROKERS", "127.0.0.1:9092"),
		HTTPAddr:           sharedconfig.String("PARSER_HTTP_ADDR", ":8081"),
		StorageDriver:      sharedconfig.String("PARSER_STORAGE_DRIVER", "postgres"),
		PostgresDSN:        sharedconfig.String("PARSER_POSTGRES_DSN", "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable"),
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
		DzenRSSFeedURL:     sharedconfig.String("DZEN_RSS_FEED_URL", ""),
		YandexRSSFeedURL:   sharedconfig.String("YANDEX_RSS_FEED_URL", ""),
		CustomRSSSources:   sharedconfig.String("CUSTOM_RSS_SOURCES", ""),
		DisabledSources:    sharedconfig.String("PARSER_DISABLED_SOURCES", ""),
	}
}

type RSSSource struct {
	Name        string
	DisplayName string
	URL         string
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
	sources := make([]RSSSource, 0)
	if strings.TrimSpace(cfg.DzenRSSFeedURL) != "" {
		sources = append(sources, RSSSource{Name: "dzen", DisplayName: "Dzen", URL: strings.TrimSpace(cfg.DzenRSSFeedURL)})
	}
	if strings.TrimSpace(cfg.YandexRSSFeedURL) != "" {
		sources = append(sources, RSSSource{Name: "yandex", DisplayName: "Yandex", URL: strings.TrimSpace(cfg.YandexRSSFeedURL)})
	}
	parts := strings.Split(cfg.CustomRSSSources, ",")
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
		sources = append(sources, RSSSource{Name: name, DisplayName: name, URL: sourceURL})
	}
	return sources
}

func (cfg Config) DisabledSourceSet() map[string]bool {
	parts := strings.Split(cfg.DisabledSources, ",")
	disabled := make(map[string]bool, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			disabled[name] = true
		}
	}
	return disabled
}

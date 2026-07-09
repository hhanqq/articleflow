package config

import (
	"strings"
	"time"

	sharedconfig "github.com/hanq/articleflow/packages/config"
	"github.com/hanq/articleflow/services/parser-service/internal/scheduler"
)

type Config struct {
	ServiceName         string
	KafkaBrokers        string
	HTTPAddr            string
	StorageDriver       string
	PostgresDSN         string
	HabrBaseURL         string
	HabrMaxAttempts     int
	HabrRetryDelayMS    int
	HabrRequestDelayMS  int
	VCBaseURL           string
	VCRSSFeedURL        string
	VCSearchProvider    string
	GoogleSearchAPIKey  string
	GoogleSearchCX      string
	BingSearchAPIKey    string
	DzenBaseURL         string
	DzenRSSFeedURL      string
	YandexRSSFeedURL    string
	CustomRSSSources    string
	DisabledSources     string
	SchedulerEnabled    bool
	SchedulerQueries    string
	SchedulerSources    string
	SchedulerLimit      int
	SchedulerIntervalMS int
}

func Load() Config {
	return Config{
		ServiceName:         "parser-service",
		KafkaBrokers:        sharedconfig.String("KAFKA_BROKERS", "127.0.0.1:9092"),
		HTTPAddr:            sharedconfig.String("PARSER_HTTP_ADDR", ":8081"),
		StorageDriver:       sharedconfig.String("PARSER_STORAGE_DRIVER", "postgres"),
		PostgresDSN:         sharedconfig.String("PARSER_POSTGRES_DSN", "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable"),
		HabrBaseURL:         sharedconfig.String("HABR_BASE_URL", "https://habr.com"),
		HabrMaxAttempts:     sharedconfig.Int("HABR_MAX_ATTEMPTS", 3),
		HabrRetryDelayMS:    sharedconfig.Int("HABR_RETRY_DELAY_MS", 500),
		HabrRequestDelayMS:  sharedconfig.Int("HABR_REQUEST_DELAY_MS", 500),
		VCBaseURL:           sharedconfig.String("VC_BASE_URL", "https://vc.ru"),
		VCRSSFeedURL:        sharedconfig.String("VC_RSS_FEED_URL", "https://vc.ru/rss"),
		VCSearchProvider:    sharedconfig.String("VC_SEARCH_PROVIDER", "discovery"),
		GoogleSearchAPIKey:  sharedconfig.String("GOOGLE_SEARCH_API_KEY", ""),
		GoogleSearchCX:      sharedconfig.String("GOOGLE_SEARCH_CX", ""),
		BingSearchAPIKey:    sharedconfig.String("BING_SEARCH_API_KEY", ""),
		DzenBaseURL:         sharedconfig.String("DZEN_BASE_URL", "https://dzen.ru"),
		DzenRSSFeedURL:      sharedconfig.String("DZEN_RSS_FEED_URL", ""),
		YandexRSSFeedURL:    sharedconfig.String("YANDEX_RSS_FEED_URL", ""),
		CustomRSSSources:    sharedconfig.String("CUSTOM_RSS_SOURCES", ""),
		DisabledSources:     sharedconfig.String("PARSER_DISABLED_SOURCES", ""),
		SchedulerEnabled:    parseBool(sharedconfig.String("PARSER_SCHEDULER_ENABLED", "false")),
		SchedulerQueries:    sharedconfig.String("PARSER_SCHEDULER_QUERIES", ""),
		SchedulerSources:    sharedconfig.String("PARSER_SCHEDULER_SOURCES", "habr,vc,dzen"),
		SchedulerLimit:      sharedconfig.Int("PARSER_SCHEDULER_LIMIT", 80),
		SchedulerIntervalMS: sharedconfig.Int("PARSER_SCHEDULER_INTERVAL_MS", 1800000),
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
		sources = append(sources, RSSSource{Name: "dzen_rss", DisplayName: "Dzen RSS", URL: strings.TrimSpace(cfg.DzenRSSFeedURL)})
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

func (cfg Config) SchedulerConfig() scheduler.Config {
	queries := make([]scheduler.Query, 0)
	sources := splitCSV(cfg.SchedulerSources)
	for _, queryText := range strings.Split(cfg.SchedulerQueries, ";") {
		queryText = strings.TrimSpace(queryText)
		if queryText == "" {
			continue
		}
		queries = append(queries, scheduler.Query{
			Text:    queryText,
			Sources: sources,
			Limit:   cfg.SchedulerLimit,
		})
	}
	interval := time.Duration(cfg.SchedulerIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return scheduler.Config{
		Enabled:  cfg.SchedulerEnabled,
		Queries:  queries,
		Interval: interval,
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		result = append(result, part)
	}
	return result
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

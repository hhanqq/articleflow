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
	VCRSSFeedURL       string
}

func Load() Config {
	return Config{
		ServiceName:        "parser-service",
		KafkaBrokers:       sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
		HTTPAddr:           sharedconfig.String("PARSER_HTTP_ADDR", ":8081"),
		HabrBaseURL:        sharedconfig.String("HABR_BASE_URL", "https://habr.com"),
		HabrMaxAttempts:    sharedconfig.Int("HABR_MAX_ATTEMPTS", 3),
		HabrRetryDelayMS:   sharedconfig.Int("HABR_RETRY_DELAY_MS", 500),
		HabrRequestDelayMS: sharedconfig.Int("HABR_REQUEST_DELAY_MS", 500),
		VCRSSFeedURL:       sharedconfig.String("VC_RSS_FEED_URL", "https://vc.ru/rss"),
	}
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

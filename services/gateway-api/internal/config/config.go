package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName         string
	HTTPAddr            string
	ArticleServiceURL   string
	ParserServiceURL    string
	FeedServiceURL      string
	KafkaBrokers        string
	RateLimitPerMinute  int
	FeedCacheTTLSeconds int
}

func Load() Config {
	return Config{
		ServiceName:         "gateway-api",
		HTTPAddr:            sharedconfig.String("GATEWAY_HTTP_ADDR", ":8080"),
		ArticleServiceURL:   sharedconfig.String("ARTICLE_SERVICE_URL", "http://localhost:8083"),
		ParserServiceURL:    sharedconfig.String("PARSER_SERVICE_URL", "http://localhost:8081"),
		FeedServiceURL:      sharedconfig.String("FEED_SERVICE_URL", "http://localhost:8082"),
		KafkaBrokers:        sharedconfig.String("KAFKA_BROKERS", "127.0.0.1:9092"),
		RateLimitPerMinute:  sharedconfig.Int("GATEWAY_RATE_LIMIT_PER_MINUTE", 120),
		FeedCacheTTLSeconds: sharedconfig.Int("GATEWAY_FEED_CACHE_TTL_SECONDS", 15),
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

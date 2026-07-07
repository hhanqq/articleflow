package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName      string
	HTTPAddr         string
	ParserServiceURL string
	FeedServiceURL   string
	KafkaBrokers     string
}

func Load() Config {
	return Config{
		ServiceName:      "gateway-api",
		HTTPAddr:         sharedconfig.String("GATEWAY_HTTP_ADDR", ":8080"),
		ParserServiceURL: sharedconfig.String("PARSER_SERVICE_URL", "http://localhost:8081"),
		FeedServiceURL:   sharedconfig.String("FEED_SERVICE_URL", "http://localhost:8082"),
		KafkaBrokers:     sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
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

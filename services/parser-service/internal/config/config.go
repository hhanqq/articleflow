package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName  string
	KafkaBrokers string
}

func Load() Config {
	return Config{
		ServiceName:  "parser-service",
		KafkaBrokers: sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
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

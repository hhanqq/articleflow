package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName             string
	GRPCAddr                string
	HTTPAddr                string
	KafkaBrokers            string
	FeedScoredTopic         string
	FeedConsumerGroupID     string
	FeedConsumerMaxMessages int
	StorageDriver           string
	PostgresDSN             string
}

func Load() Config {
	return Config{
		ServiceName:             "feed-service",
		GRPCAddr:                sharedconfig.String("FEED_GRPC_ADDR", ":9002"),
		HTTPAddr:                sharedconfig.String("FEED_HTTP_ADDR", ":8082"),
		KafkaBrokers:            sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
		FeedScoredTopic:         sharedconfig.String("FEED_SCORED_TOPIC", "feed.item.scored.v1"),
		FeedConsumerGroupID:     sharedconfig.String("FEED_CONSUMER_GROUP_ID", "feed-service"),
		FeedConsumerMaxMessages: sharedconfig.Int("FEED_CONSUMER_MAX_MESSAGES", 0),
		StorageDriver:           sharedconfig.String("FEED_STORAGE_DRIVER", "postgres"),
		PostgresDSN:             sharedconfig.String("FEED_POSTGRES_DSN", "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable"),
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

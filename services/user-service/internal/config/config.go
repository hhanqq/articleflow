package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName             string
	GRPCAddr                string
	KafkaBrokers            string
	UserReactionTopic       string
	UserConsumerGroupID     string
	UserConsumerMaxMessages int
	StorageDriver           string
	PostgresDSN             string
}

func Load() Config {
	return Config{
		ServiceName:             "user-service",
		GRPCAddr:                sharedconfig.String("USER_GRPC_ADDR", ":9003"),
		KafkaBrokers:            sharedconfig.String("KAFKA_BROKERS", "127.0.0.1:9092"),
		UserReactionTopic:       sharedconfig.String("USER_REACTION_TOPIC", "user.reaction.created.v1"),
		UserConsumerGroupID:     sharedconfig.String("USER_CONSUMER_GROUP_ID", "user-service"),
		UserConsumerMaxMessages: sharedconfig.Int("USER_CONSUMER_MAX_MESSAGES", 0),
		StorageDriver:           sharedconfig.String("USER_STORAGE_DRIVER", "postgres"),
		PostgresDSN:             sharedconfig.String("USER_POSTGRES_DSN", "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable"),
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

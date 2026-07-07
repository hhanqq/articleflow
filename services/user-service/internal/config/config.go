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
}

func Load() Config {
	return Config{
		ServiceName:             "user-service",
		GRPCAddr:                sharedconfig.String("USER_GRPC_ADDR", ":9003"),
		KafkaBrokers:            sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
		UserReactionTopic:       sharedconfig.String("USER_REACTION_TOPIC", "user.reaction.created.v1"),
		UserConsumerGroupID:     sharedconfig.String("USER_CONSUMER_GROUP_ID", "user-service"),
		UserConsumerMaxMessages: sharedconfig.Int("USER_CONSUMER_MAX_MESSAGES", 0),
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

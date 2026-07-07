package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName                string
	GRPCAddr                   string
	KafkaBrokers               string
	ArticleDiscoveredTopic     string
	RankingConsumerGroupID     string
	RankingConsumerMaxMessages int
}

func Load() Config {
	return Config{
		ServiceName:                "ranking-service",
		GRPCAddr:                   sharedconfig.String("RANKING_GRPC_ADDR", ":9004"),
		KafkaBrokers:               sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
		ArticleDiscoveredTopic:     sharedconfig.String("ARTICLE_DISCOVERED_TOPIC", "article.discovered.v1"),
		RankingConsumerGroupID:     sharedconfig.String("RANKING_CONSUMER_GROUP_ID", "ranking-service"),
		RankingConsumerMaxMessages: sharedconfig.Int("RANKING_CONSUMER_MAX_MESSAGES", 0),
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

package config

import (
	"strings"

	sharedconfig "github.com/hanq/articleflow/packages/config"
)

type Config struct {
	ServiceName            string
	HTTPAddr               string
	GRPCAddr               string
	KafkaBrokers           string
	ArticleDiscoveredTopic string
	ArticleConsumerGroupID string
	ConsumerMaxMessages    int
	StorageDriver          string
	PostgresDSN            string
}

func Load() Config {
	return Config{
		ServiceName:            "article-service",
		HTTPAddr:               sharedconfig.String("ARTICLE_HTTP_ADDR", ":8083"),
		GRPCAddr:               sharedconfig.String("ARTICLE_GRPC_ADDR", ":9001"),
		KafkaBrokers:           sharedconfig.String("KAFKA_BROKERS", "localhost:9092"),
		ArticleDiscoveredTopic: sharedconfig.String("ARTICLE_DISCOVERED_TOPIC", "article.discovered.v1"),
		ArticleConsumerGroupID: sharedconfig.String("ARTICLE_CONSUMER_GROUP_ID", "article-service"),
		ConsumerMaxMessages:    sharedconfig.Int("ARTICLE_CONSUMER_MAX_MESSAGES", 0),
		StorageDriver:          sharedconfig.String("ARTICLE_STORAGE_DRIVER", "postgres"),
		PostgresDSN:            sharedconfig.String("ARTICLE_POSTGRES_DSN", "postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable"),
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

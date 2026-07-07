package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "article-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.HTTPAddr != ":8083" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != ":9001" {
		t.Fatalf("unexpected gRPC addr: %s", cfg.GRPCAddr)
	}
	if cfg.KafkaBrokers != "localhost:9092" {
		t.Fatalf("unexpected Kafka brokers: %s", cfg.KafkaBrokers)
	}
	if cfg.ArticleDiscoveredTopic != "article.discovered.v1" {
		t.Fatalf("unexpected topic: %s", cfg.ArticleDiscoveredTopic)
	}
	if cfg.ArticleConsumerGroupID != "article-service" {
		t.Fatalf("unexpected consumer group: %s", cfg.ArticleConsumerGroupID)
	}
	if cfg.StorageDriver != "postgres" {
		t.Fatalf("unexpected storage driver: %s", cfg.StorageDriver)
	}
	if cfg.ConsumerMaxMessages != 0 {
		t.Fatalf("unexpected max messages: %d", cfg.ConsumerMaxMessages)
	}
	if cfg.PostgresDSN == "" {
		t.Fatal("expected default Postgres DSN")
	}
}

func TestBrokerListSplitsCommaSeparatedBrokers(t *testing.T) {
	cfg := Config{KafkaBrokers: "localhost:9092, kafka:29092 ,"}

	brokers := cfg.BrokerList()

	if len(brokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(brokers))
	}
	if brokers[0] != "localhost:9092" || brokers[1] != "kafka:29092" {
		t.Fatalf("unexpected brokers: %#v", brokers)
	}
}

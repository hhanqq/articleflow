package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "parser-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.KafkaBrokers != "localhost:9092" {
		t.Fatalf("unexpected Kafka brokers: %s", cfg.KafkaBrokers)
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

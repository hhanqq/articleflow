package config

import "testing"

func TestBrokerListTrimsEmptyParts(t *testing.T) {
	cfg := Config{KafkaBrokers: " localhost:9092, kafka:9092, "}

	brokers := cfg.BrokerList()

	if len(brokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(brokers))
	}
	if brokers[0] != "localhost:9092" || brokers[1] != "kafka:9092" {
		t.Fatalf("unexpected brokers: %#v", brokers)
	}
}

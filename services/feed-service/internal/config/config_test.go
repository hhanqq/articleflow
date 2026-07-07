package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "feed-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.GRPCAddr != ":9002" {
		t.Fatalf("unexpected gRPC addr: %s", cfg.GRPCAddr)
	}
	if cfg.HTTPAddr != ":8082" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTPAddr)
	}
	if cfg.FeedScoredTopic != "feed.item.scored.v1" {
		t.Fatalf("unexpected scored topic: %s", cfg.FeedScoredTopic)
	}
	if cfg.StorageDriver != "postgres" {
		t.Fatalf("unexpected storage driver: %s", cfg.StorageDriver)
	}
	if cfg.PostgresDSN == "" {
		t.Fatal("expected default postgres dsn")
	}
}

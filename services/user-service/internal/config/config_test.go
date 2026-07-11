package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "user-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.HTTPAddr != ":8084" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != ":9003" {
		t.Fatalf("unexpected gRPC addr: %s", cfg.GRPCAddr)
	}
	if cfg.UserReactionTopic != "user.reaction.created.v1" {
		t.Fatalf("unexpected reaction topic: %s", cfg.UserReactionTopic)
	}
	if cfg.UserConsumerGroupID != "user-service" {
		t.Fatalf("unexpected consumer group: %s", cfg.UserConsumerGroupID)
	}
	if cfg.StorageDriver != "postgres" {
		t.Fatalf("unexpected storage driver: %s", cfg.StorageDriver)
	}
	if cfg.PostgresDSN == "" {
		t.Fatal("expected postgres dsn")
	}
}

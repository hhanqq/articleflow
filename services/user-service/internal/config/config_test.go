package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "user-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
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
}

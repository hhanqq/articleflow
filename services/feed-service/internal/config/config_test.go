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
}


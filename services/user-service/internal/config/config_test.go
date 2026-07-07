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
}


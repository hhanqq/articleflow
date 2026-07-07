package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "gateway-api" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTPAddr)
	}
}


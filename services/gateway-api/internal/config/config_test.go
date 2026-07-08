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
	if cfg.ParserServiceURL != "http://localhost:8081" {
		t.Fatalf("unexpected parser service URL: %s", cfg.ParserServiceURL)
	}
	if cfg.ArticleServiceURL != "http://localhost:8083" {
		t.Fatalf("unexpected article service URL: %s", cfg.ArticleServiceURL)
	}
	if cfg.FeedServiceURL != "http://localhost:8082" {
		t.Fatalf("unexpected feed service URL: %s", cfg.FeedServiceURL)
	}
	if cfg.KafkaBrokers != "127.0.0.1:9092" {
		t.Fatalf("unexpected kafka brokers: %s", cfg.KafkaBrokers)
	}
}

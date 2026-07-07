package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "ranking-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.GRPCAddr != ":9004" {
		t.Fatalf("unexpected gRPC addr: %s", cfg.GRPCAddr)
	}
	if cfg.ArticleDiscoveredTopic != "article.discovered.v1" {
		t.Fatalf("unexpected discovered topic: %s", cfg.ArticleDiscoveredTopic)
	}
	if cfg.RankingConsumerGroupID != "ranking-service" {
		t.Fatalf("unexpected consumer group: %s", cfg.RankingConsumerGroupID)
	}
}

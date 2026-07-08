package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.ServiceName != "parser-service" {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
	if cfg.KafkaBrokers != "127.0.0.1:9092" {
		t.Fatalf("unexpected Kafka brokers: %s", cfg.KafkaBrokers)
	}
	if cfg.HTTPAddr != ":8081" {
		t.Fatalf("unexpected HTTP addr: %s", cfg.HTTPAddr)
	}
	if cfg.HabrBaseURL != "https://habr.com" {
		t.Fatalf("unexpected Habr base URL: %s", cfg.HabrBaseURL)
	}
	if cfg.HabrMaxAttempts != 3 {
		t.Fatalf("unexpected Habr max attempts: %d", cfg.HabrMaxAttempts)
	}
	if cfg.HabrRetryDelayMS != 500 {
		t.Fatalf("unexpected Habr retry delay: %d", cfg.HabrRetryDelayMS)
	}
	if cfg.HabrRequestDelayMS != 500 {
		t.Fatalf("unexpected Habr request delay: %d", cfg.HabrRequestDelayMS)
	}
	if cfg.VCRSSFeedURL != "https://vc.ru/rss" {
		t.Fatalf("unexpected vc.ru RSS URL: %s", cfg.VCRSSFeedURL)
	}
	if cfg.VCSearchProvider != "" {
		t.Fatalf("unexpected vc search provider: %s", cfg.VCSearchProvider)
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

func TestRSSSourceListParsesConfiguredSources(t *testing.T) {
	cfg := Config{CustomRSSSources: "dzen=https://dzen.ru/rss, yandex=https://news.yandex.ru/index.rss , broken"}

	sources := cfg.RSSSourceList()

	if len(sources) != 2 {
		t.Fatalf("expected 2 rss sources, got %d", len(sources))
	}
	if sources[0].Name != "dzen" || sources[0].URL != "https://dzen.ru/rss" {
		t.Fatalf("unexpected first source: %#v", sources[0])
	}
	if sources[1].Name != "yandex" || sources[1].URL != "https://news.yandex.ru/index.rss" {
		t.Fatalf("unexpected second source: %#v", sources[1])
	}
}

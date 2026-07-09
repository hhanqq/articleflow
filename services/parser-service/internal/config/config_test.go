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
	if cfg.StorageDriver != "postgres" {
		t.Fatalf("unexpected storage driver: %s", cfg.StorageDriver)
	}
	if cfg.PostgresDSN == "" {
		t.Fatal("expected default postgres dsn")
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
	if cfg.VCSearchProvider != "discovery" {
		t.Fatalf("unexpected vc search provider: %s", cfg.VCSearchProvider)
	}
	if cfg.DzenBaseURL != "https://dzen.ru" {
		t.Fatalf("unexpected dzen base URL: %s", cfg.DzenBaseURL)
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
	cfg := Config{
		DzenRSSFeedURL:   "https://dzen.example/rss",
		YandexRSSFeedURL: "https://yandex.example/rss",
		CustomRSSSources: "custom=https://custom.example/rss, broken",
	}

	sources := cfg.RSSSourceList()

	if len(sources) != 3 {
		t.Fatalf("expected 3 rss sources, got %d", len(sources))
	}
	if sources[0].Name != "dzen_rss" || sources[0].DisplayName != "Dzen RSS" || sources[0].URL != "https://dzen.example/rss" {
		t.Fatalf("unexpected first source: %#v", sources[0])
	}
	if sources[1].Name != "yandex" || sources[1].DisplayName != "Yandex" || sources[1].URL != "https://yandex.example/rss" {
		t.Fatalf("unexpected second source: %#v", sources[1])
	}
	if sources[2].Name != "custom" || sources[2].DisplayName != "custom" || sources[2].URL != "https://custom.example/rss" {
		t.Fatalf("unexpected custom source: %#v", sources[2])
	}
}

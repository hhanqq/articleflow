# parser-service

Fetches external article sources and publishes `article.discovered.v1` events to Kafka.

Current sources:

- `habr`: Habr RSS search plus full article HTML parsing
- `vc`: generic RSS parser pointed at `https://vc.ru/rss`

Run:

```bash
go run ./cmd/parser-service
```

Useful env:

- `HABR_BASE_URL` defaults to `https://habr.com`
- `VC_RSS_FEED_URL` defaults to `https://vc.ru/rss`
- `KAFKA_BROKERS` defaults to `localhost:9092`

# parser-service

Fetches external article sources and publishes `article.discovered.v1` events to Kafka.

Current sources:

- `habr`: Habr RSS search plus full article HTML parsing
- `vc`: safe vc.ru source backed by RSS until a stable official/search integration is added
- `vc_rss`: generic RSS parser pointed at `https://vc.ru/rss`
- custom RSS sources from `CUSTOM_RSS_SOURCES`, for example `dzen=https://dzen.ru/rss,yandex=https://news.yandex.ru/index.rss`

Run:

```bash
go run ./cmd/parser-service
```

Useful env:

- `HABR_BASE_URL` defaults to `https://habr.com`
- `VC_BASE_URL` defaults to `https://vc.ru`
- `VC_RSS_FEED_URL` defaults to `https://vc.ru/rss`
- `CUSTOM_RSS_SOURCES` is a comma-separated source registry in `name=url` format
- `KAFKA_BROKERS` defaults to `127.0.0.1:9092`

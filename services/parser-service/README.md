# parser-service

Fetches external article sources and publishes `article.discovered.v1` events to Kafka.

Current sources:

- `habr`: Habr RSS search plus full article HTML parsing
- `vc`: full-site vc.ru search through configured official search provider, then public article HTML parsing through `window.__INITIAL_STATE__`, JSON-LD, and meta tags; falls back to RSS if no provider is configured
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
- `VC_SEARCH_PROVIDER` can be `google`, `google_cse`, or `bing`
- `GOOGLE_SEARCH_API_KEY` and `GOOGLE_SEARCH_CX` enable Google Programmable Search for `vc`
- `BING_SEARCH_API_KEY` enables Bing Web Search for `vc`
- `CUSTOM_RSS_SOURCES` is a comma-separated source registry in `name=url` format
- `KAFKA_BROKERS` defaults to `127.0.0.1:9092`

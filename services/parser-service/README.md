# parser-service

Fetches external article sources and publishes `article.discovered.v1` events to Kafka.

Current sources:

- `habr`: Habr RSS search plus full article HTML parsing
- `vc`: full-site vc.ru search through the public discovery API used by `https://vc.ru/discovery?q=...`, then public article HTML parsing through `window.__INITIAL_STATE__`, JSON-LD, and meta tags; falls back to RSS when discovery is empty or temporarily unavailable
- `dzen`: Dzen HTML search through `/search?query=...`, then article HTML parsing through JSON-LD, meta tags, and `<article>` text
- `vc_rss`: generic RSS parser pointed at `https://vc.ru/rss`
- optional Dzen/Yandex RSS or Atom feeds via `DZEN_RSS_FEED_URL` and `YANDEX_RSS_FEED_URL`
- custom RSS/Atom sources from `CUSTOM_RSS_SOURCES`, for example `my_blog=https://example.com/feed.xml`

Run:

```bash
go run ./cmd/parser-service
```

Useful env:

- `HABR_BASE_URL` defaults to `https://habr.com`
- `VC_BASE_URL` defaults to `https://vc.ru`
- `VC_RSS_FEED_URL` defaults to `https://vc.ru/rss`
- `VC_SEARCH_PROVIDER` defaults to `discovery`; it can also be `google`, `google_cse`, or `bing`
- `GOOGLE_SEARCH_API_KEY` and `GOOGLE_SEARCH_CX` enable Google Programmable Search for `vc`
- `BING_SEARCH_API_KEY` enables Bing Web Search for `vc`
- `DZEN_BASE_URL` defaults to `https://dzen.ru`
- `DZEN_RSS_FEED_URL` adds a separate `dzen_rss` source when a concrete Dzen feed URL is known
- `YANDEX_RSS_FEED_URL` adds a `yandex` source when a concrete Yandex feed URL is known
- `CUSTOM_RSS_SOURCES` is a comma-separated source registry in `name=url` format
- `PARSER_DISABLED_SOURCES` disables registered sources by name, for example `vc_rss,dzen`
- `KAFKA_BROKERS` defaults to `127.0.0.1:9092`

For `vc` discovery search, the first vc.ru API page is usually 12 items. Use parser job `limit` to fetch more pages; `SearchQuery` caps it at 100 per job.

For `dzen`, direct server-side requests to `https://dzen.ru/search?query=...` can be redirected to Yandex/VK SSO. In that case parser-service records a failed `dzen` source stat with the auth redirect URL instead of masking it as a Kafka or job infrastructure error.

When a job searches multiple sources, parser-service deduplicates candidates and interleaves returned results by source before applying `limit`. Empty `sources` means all registered sources.

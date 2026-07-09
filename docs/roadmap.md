# Articleflow Roadmap

## Product stages

1. Feed refill: query feed reads from the article DB first, then starts parser jobs in the background when the stored pool is thin.
2. Cursor pagination: feed APIs expose cursor tokens for scrolling; the first implementation uses `offset:N` tokens.
3. Source strategy stats: every source reports the strategy used, such as HTML search, HTML/JSON search, or RSS fallback.
4. Anti-duplicates and canonical quality: canonical URLs, similar titles, and source aliases prevent duplicate feed cards.
5. Ranking v2: query relevance, freshness, source diversity, user signals, and explicit score reasons.
6. Personalization: user reactions and future opens/skips/saves influence tag and source weights.
7. UI product feed: vertical scrolling feed, compact controls, refill status, and source diagnostics.
8. Scheduler and background ingestion: recurring parser jobs keep common topics and configured sources warm.
9. Observability: parser/source latency, accepted/filtered counts, feed latency, Kafka lag, and error dashboards.
10. Production hardening: env templates, rate limits, cache, migrations, load limits, and graceful source degradation.

## Historical implementation stages

1. Runtime service wiring: connect parser, Kafka, article-service consumers, env config, graceful shutdown.
2. Postgres persistence: repositories, migrations, runtime storage selection, health checks.
3. Full Habr parser: search jobs, full article parsing, rate limits, retries, parser failure events.
4. Search pipeline: stored article search, gateway search jobs, parser-service execution, job status endpoints.
5. Ranking and feed pipeline: ranking events, ranked feed storage, reaction-based scoring.
6. Frontend SPA: vertical article feed, search screen, reactions, parser job status.
7. Observability and production readiness: logs, metrics, traces, Dockerfiles, e2e commands.

## Current status

Stages 1 and 2 are implemented and verified by `make e2e-article-chain`:

```text
Kafka article.discovered.v1
  -> article-service runtime consumer loop
  -> article-service discovered handler
  -> article-service ingest usecase
  -> Postgres article store
```

Stage 3 is implemented: `parser-service/cmd/search-habr` can search Habr through RSS, fetch full article HTML pages, publish full-content `article.discovered.v1` events, and publish `parser.job.failed.v1` on parser source failure.

Stage 4 is implemented: stored articles are searchable through gateway `/api/v1/search` backed by article-service/Postgres full-text search. It supports source, tag, date, limit, and offset filters. Parser jobs can be created and checked through parser-service HTTP endpoints and gateway search job proxy endpoints. Completed parser jobs include candidate payloads, not only counts, and fresh parser candidates are normalized and deduplicated before Kafka publish.

Stage 5 is implemented: ranking-service consumes `article.discovered.v1`, publishes `feed.item.scored.v1`, feed-service stores a ranked in-memory read model, and gateway reads `/api/v1/feed` through feed-service.

Stage 6 is implemented: `web/` contains a dependency-free static SPA for the vertical feed, stored search, separate parser job creation/status polling, API base settings, and article reactions.

Stage 7 is implemented: gateway has CORS, gateway/parser/feed/article expose `/metrics`, observability has a small Prometheus-style metrics registry and HTTP request counter middleware, Dockerfiles are available for Go services and web, `make ci` runs the fast verification gate in GitHub Actions, and `make e2e-ranking-feed` verifies the ranking/feed runtime chain locally.

Feed persistence is available: feed-service can use memory storage by default or Postgres through `FEED_STORAGE_DRIVER=postgres`, backed by the `feed_items` migration.

## Current product focus

The active product track is stages 2, 5, 6, and 8:

- Stage 2 adds cursor pagination to query feed responses so the UI can scroll without offset-only behavior.
- Stage 5 improves ranking with freshness, relevance, source diversity, and visible score reasons.
- Stage 6 extends personalization through reaction-derived tag/source weights.
- Stage 8 adds background ingestion scheduling so parser jobs can warm the shared article pool without a user click.

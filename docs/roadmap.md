# Articleflow Roadmap

## Seven implementation stages

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

Stage 4 is implemented: stored articles are searchable through gateway `/api/v1/search` backed by article-service/Postgres, and parser jobs can be created and checked through parser-service HTTP endpoints and gateway search job proxy endpoints. Completed parser jobs include candidate payloads, not only counts.

Stage 5 is implemented: ranking-service consumes `article.discovered.v1`, publishes `feed.item.scored.v1`, feed-service stores a ranked in-memory read model, and gateway reads `/api/v1/feed` through feed-service.

Stage 6 is implemented: `web/` contains a dependency-free static SPA for the vertical feed, stored search, separate parser job creation/status polling, API base settings, and article reactions.

Stage 7 is implemented: gateway has CORS and `/metrics`, observability has a small Prometheus-style metrics registry, Dockerfiles are available for Go services and web, and `make e2e-ranking-feed` verifies the ranking/feed runtime chain.

Feed persistence is available: feed-service can use memory storage by default or Postgres through `FEED_STORAGE_DRIVER=postgres`, backed by the `feed_items` migration.

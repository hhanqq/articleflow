# Articleflow Roadmap

## Seven implementation stages

1. Runtime service wiring: connect parser, Kafka, article-service consumers, env config, graceful shutdown.
2. Postgres persistence: repositories, migrations, runtime storage selection, health checks.
3. Full Habr parser: search jobs, full article parsing, rate limits, retries, parser failure events.
4. Search pipeline: gateway search jobs, parser-service execution, job status endpoints.
5. Ranking and feed pipeline: ranking events, ranked feed storage, reaction-based scoring.
6. Frontend SPA: vertical article feed, search screen, reactions, parser job status.
7. Observability and production readiness: logs, metrics, traces, Dockerfiles, e2e commands.

## Current focus

Implement stages 1 and 2 enough to run the first backend chain:

```text
Kafka article.discovered.v1
  -> article-service runtime consumer loop
  -> article-service discovered handler
  -> article-service ingest usecase
  -> Postgres article store
```

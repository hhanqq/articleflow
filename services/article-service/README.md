# article-service

Owns article storage, deduplication, and article detail API.

Runtime:

- consumes Kafka topic `article.discovered.v1`
- stores full article payloads in memory or Postgres
- serves `GET /api/v1/articles?id=<article_id>`
- serves `GET /healthz`

Run:

```bash
go run ./cmd/article-service
```

Useful env:

- `ARTICLE_HTTP_ADDR` defaults to `:8083`
- `ARTICLE_STORAGE_DRIVER` defaults to `memory`; use `postgres` for shared dev data
- `ARTICLE_POSTGRES_DSN` defaults to local Docker Compose Postgres
- `ARTICLE_CONSUMER_GROUP_ID` defaults to `article-service`

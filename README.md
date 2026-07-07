# Articleflow

Articleflow is a microservice-first Go project for a TikTok-like vertical feed of articles from Habr, VC.ru, Yandex/Dzen, RSS feeds, and future sources.

## Local Go Isolation

The project does not use global Go caches or global tool installation during normal development. Use `make` targets or source `scripts/env.sh`:

```bash
source scripts/env.sh
make test
```

Local paths:

```text
.go/pkg/mod      Go module cache
.cache/go-build Go build cache
.bin            Go tool binaries
```

## Architecture

- `gateway-api`: REST/JSON API for frontend clients.
- `parser-service`: source parsers and Kafka article discovery producer.
- `article-service`: article storage and deduplication.
- `feed-service`: feed read model.
- `user-service`: users, reactions, saved articles.
- `ranking-service`: scoring and ranking.
- `contracts`: protobuf and Go DTO/event contracts.
- `packages`: shared infrastructure helpers.

## Current Event Flow

The first tested vertical slice is in place:

```text
Habr RSS XML / Habr HTML search
  -> parser-service/internal/parsers/habr
  -> parser-service/internal/usecase.DiscoveredPublisher
  -> Kafka topic article.discovered.v1
  -> article-service/internal/transport/kafka.DiscoveredHandler
  -> article-service/internal/usecase.IngestUsecase
```

Kafka adapters live in `packages/kafka`. Unit tests use the in-memory producer; runtime wiring uses `segmentio/kafka-go`.
`article-service/internal/runtime.ConsumerLoop` is the testable Kafka consumer loop used to bridge a runtime consumer and a message handler.

`article-service` now has runtime wiring for:

```text
Kafka article.discovered.v1
  -> ReaderConsumer
  -> ConsumerLoop with commit after successful handling
  -> DiscoveredHandler
  -> IngestUsecase
  -> memory or Postgres ArticleStore
```

Postgres schema is available both as a migration file and as Docker init SQL:

```text
deployments/postgres/migrations/001_articles.sql
deployments/postgres/init/002_articles.sql
```

Stage 3 Habr runtime search is available through `services/parser-service/cmd/search-habr`.
It searches Habr, fetches each full article page, publishes `article.discovered.v1` with full `content`, and publishes `parser.job.failed.v1` when a parser source fails.

Stage 4 search jobs are available through parser-service HTTP endpoints and gateway proxy endpoints.

## Gateway API

Default address: `:8080`. Parser-service upstream defaults to `http://localhost:8081`.

```bash
source scripts/env.sh
cd services/gateway-api
go run ./cmd/gateway-api
```

Available REST endpoints:

```text
GET  /healthz
GET  /api/v1/feed?limit=20
POST /api/v1/search
POST /api/v1/search/jobs
GET  /api/v1/search/jobs/{id}
POST /api/v1/reactions
```

Example search request:

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"go kafka","sources":["habr"],"limit":10}'
```

Example async parser job through gateway:

```bash
curl -X POST http://localhost:8080/api/v1/search/jobs \
  -H 'Content-Type: application/json' \
  -d '{"query":"go kafka","sources":["habr"],"limit":5}'
```

Check job status:

```bash
curl http://localhost:8080/api/v1/search/jobs/<job-id>
```

Example reaction request:

```bash
curl -X POST http://localhost:8080/api/v1/reactions \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-1","article_id":"article-1","type":"save"}'
```

## Article Service Runtime

Default mode uses in-memory storage:

```bash
source scripts/env.sh
cd services/article-service
go run ./cmd/article-service
```

Run against local Docker Postgres:

```bash
docker compose -f deployments/docker-compose.yml up -d postgres kafka zookeeper

source scripts/env.sh
cd services/article-service
ARTICLE_STORAGE_DRIVER=postgres \
ARTICLE_POSTGRES_DSN='postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable' \
KAFKA_BROKERS=localhost:9092 \
ARTICLE_DISCOVERED_TOPIC=article.discovered.v1 \
ARTICLE_CONSUMER_GROUP_ID=article-service \
go run ./cmd/article-service
```

Useful runtime env:

```text
KAFKA_BROKERS=localhost:9092
ARTICLE_DISCOVERED_TOPIC=article.discovered.v1
ARTICLE_CONSUMER_GROUP_ID=article-service
ARTICLE_CONSUMER_MAX_MESSAGES=0
ARTICLE_STORAGE_DRIVER=memory
ARTICLE_POSTGRES_DSN=postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable
PARSER_HTTP_ADDR=:8081
PARSER_SERVICE_URL=http://localhost:8081
```

Publish a sample `article.discovered.v1` event:

```bash
source scripts/env.sh
cd services/parser-service
KAFKA_BROKERS=localhost:9092 go run ./cmd/publish-discovered
```

Run a real Habr search and publish discovered articles:

```bash
source scripts/env.sh
cd services/parser-service
KAFKA_BROKERS=localhost:9092 \
HABR_REQUEST_DELAY_MS=500 \
go run ./cmd/search-habr --query "go kafka" --limit 5
```

The same command is exposed via Make:

```bash
QUERY="go kafka" LIMIT=5 make search-habr
```

## Parser Service HTTP API

Default address: `:8081`.

```bash
source scripts/env.sh
cd services/parser-service
PARSER_HTTP_ADDR=:8081 KAFKA_BROKERS=localhost:9092 go run ./cmd/parser-service
```

Available parser job endpoints:

```text
POST /api/v1/parser/jobs
GET  /api/v1/parser/jobs/{id}
```

Example:

```bash
curl -X POST http://localhost:8081/api/v1/parser/jobs \
  -H 'Content-Type: application/json' \
  -d '{"query":"go kafka","sources":["habr"],"limit":5}'
```

Run the local e2e check for the first backend chain:

```bash
make e2e-article-chain
```

This command starts Kafka/Postgres via Docker Compose, applies the article schema, runs `article-service` for one Kafka message, publishes a sample discovered article, and verifies the row in Postgres.

## Local Commands

```bash
make env
make test
make tidy
make tools
make proto
make publish-sample-discovered
QUERY="go kafka" LIMIT=5 make search-habr
make e2e-article-chain
docker compose -f deployments/docker-compose.yml config
docker compose -f deployments/docker-compose.yml up -d
```

Run a service:

```bash
source scripts/env.sh
cd services/gateway-api
go run ./cmd/gateway-api
```

Smoke-run all services without leaving long-running processes:

```bash
source scripts/env.sh
for svc in gateway-api parser-service article-service feed-service user-service ranking-service; do
  echo "== $svc =="
  (cd "services/$svc" && env GATEWAY_HTTP_ADDR=:0 go run "./cmd/$svc" & pid=$!; sleep 1; kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true)
done
```

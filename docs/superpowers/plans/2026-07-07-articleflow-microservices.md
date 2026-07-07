# Articleflow Microservices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the initial Articleflow microservice workspace: Kafka-based event pipeline, gRPC/protobuf contracts, REST gateway, parser/article/feed/user/ranking service skeletons, and local Docker infrastructure.

**Architecture:** Articleflow is a Go 1.25 microservice monorepo. Services communicate internally over gRPC and Kafka events, while `gateway-api` exposes REST/JSON for a future frontend. Shared protobuf contracts live in `contracts`, while reusable infrastructure helpers live in `packages`.

**Tech Stack:** Go 1.25, protobuf/gRPC, Kafka, PostgreSQL, Redis, Docker Compose, Chi HTTP router, Sarama Kafka client, pgx, zerolog.

---

### Task 1: Workspace Foundation

**Files:**
- Create: `go.work`
- Create: `README.md`
- Create: `.gitignore`
- Create: `Makefile`
- Create: `docs/architecture.md`

- [ ] Create Go workspace with modules for contracts, packages, and six services.
- [ ] Add root README with project intent and local commands.
- [ ] Add Makefile targets: `test`, `tidy`, `proto`, `up`, `down`.
- [ ] Verify: `go work sync`.

### Task 2: Contracts

**Files:**
- Create: `contracts/go.mod`
- Create: `contracts/proto/article/v1/article.proto`
- Create: `contracts/proto/feed/v1/feed.proto`
- Create: `contracts/proto/user/v1/user.proto`
- Create: `contracts/proto/events/v1/article_events.proto`
- Create: `contracts/proto/events/v1/user_events.proto`
- Create: `contracts/article/v1/article.go`
- Create: `contracts/feed/v1/feed.go`
- Create: `contracts/user/v1/user.go`
- Create: `contracts/events/v1/events.go`
- Test: `contracts/events/v1/events_test.go`

- [ ] Write failing tests for event topic constants and ArticleDiscovered payload validation.
- [ ] Add minimal Go DTO structs and event constants.
- [ ] Keep `.proto` files as source contracts; generated Go can be added after `buf` is introduced.
- [ ] Verify: `go test ./contracts/...`.

### Task 3: Shared Packages

**Files:**
- Create: `packages/config/go.mod`
- Create: `packages/config/config.go`
- Test: `packages/config/config_test.go`
- Create: `packages/logger/go.mod`
- Create: `packages/logger/logger.go`
- Create: `packages/errors/go.mod`
- Create: `packages/errors/errors.go`
- Create: `packages/kafka/go.mod`
- Create: `packages/kafka/topics.go`
- Create: `packages/grpcx/go.mod`
- Create: `packages/grpcx/server.go`

- [ ] Write failing config tests for env defaults and overrides.
- [ ] Add small reusable helpers without embedding business logic.
- [ ] Verify: `go test ./packages/...`.

### Task 4: Infrastructure

**Files:**
- Create: `deployments/docker-compose.yml`
- Create: `deployments/postgres/init/001_init.sql`
- Create: `.env.example`

- [ ] Add Kafka, Zookeeper, Postgres, Redis, and Kafka UI.
- [ ] Add databases for article/user/feed services.
- [ ] Verify: `docker compose -f deployments/docker-compose.yml config`.

### Task 5: Service Skeletons

**Files:**
- Create service modules under:
  - `services/gateway-api`
  - `services/parser-service`
  - `services/article-service`
  - `services/feed-service`
  - `services/user-service`
  - `services/ranking-service`
- Each service gets:
  - `go.mod`
  - `cmd/<service>/main.go`
  - `internal/app/app.go`
  - `internal/config/config.go`
  - `README.md`

- [ ] Write one smoke test per service config package.
- [ ] Add minimal runnable main for each service.
- [ ] Verify: `go test ./services/...` through workspace.

### Task 6: First Vertical Flow

**Files:**
- Parser: `services/parser-service/internal/parsers/habr/rss.go`
- Article: `services/article-service/internal/usecase/ingest.go`
- Feed: `services/feed-service/internal/usecase/feed.go`
- Gateway: `services/gateway-api/internal/transport/http/feed_handler.go`

- [ ] Parser produces `ArticleDiscoveredEvent` from Habr RSS XML.
- [ ] Article service validates and stores article records.
- [ ] Feed service returns latest article previews.
- [ ] Gateway exposes `GET /api/v1/feed`.
- [ ] Verify with unit tests before wiring real Kafka consumers.

### Self-Review

- Spec coverage: services, Kafka, gRPC/protobuf contracts, REST gateway, DTOs, Docker infra, and staged parser flow are covered.
- Placeholder scan: implementation details are intentionally staged; no undefined runtime behavior is required for Task 1-5.
- Type consistency: shared DTO names are `Article`, `ArticlePreview`, `FeedItem`, `UserReaction`, `ArticleDiscoveredEvent`, and `ArticleCreatedEvent`.

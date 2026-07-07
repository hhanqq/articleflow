#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deployments/docker-compose.yml"

export GOPATH="$ROOT_DIR/.go"
export GOMODCACHE="$ROOT_DIR/.go/pkg/mod"
export GOCACHE="$ROOT_DIR/.cache/go-build"
export GOBIN="$ROOT_DIR/.bin"
export PATH="$GOBIN:$PATH"

mkdir -p "$GOPATH" "$GOMODCACHE" "$GOCACHE" "$GOBIN"

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not running or is not reachable."
  echo "Start Docker Desktop and run this command again:"
  echo "  make e2e-article-chain"
  exit 1
fi

docker compose -f "$COMPOSE_FILE" up -d postgres zookeeper kafka

echo "Waiting for Kafka..."
for _ in {1..60}; do
  if docker compose -f "$COMPOSE_FILE" exec -T kafka kafka-topics --bootstrap-server kafka:29092 --list >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Ensuring Kafka topic article.discovered.v1..."
docker compose -f "$COMPOSE_FILE" exec -T kafka kafka-topics \
  --bootstrap-server kafka:29092 \
  --create \
  --if-not-exists \
  --topic article.discovered.v1 \
  --partitions 1 \
  --replication-factor 1 >/dev/null

echo "Waiting for Postgres..."
for _ in {1..30}; do
  if docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U articleflow -d articleflow >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Applying article schema..."
docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U articleflow -d articleflow < "$ROOT_DIR/deployments/postgres/migrations/001_articles.sql" >/dev/null

ARTICLE_LOG="$(mktemp)"
cleanup() {
  if [[ -n "${ARTICLE_PID:-}" ]]; then
    kill "$ARTICLE_PID" >/dev/null 2>&1 || true
    wait "$ARTICLE_PID" >/dev/null 2>&1 || true
  fi
  rm -f "$ARTICLE_LOG"
}
trap cleanup EXIT

echo "Starting article-service for one Kafka message..."
(
  cd "$ROOT_DIR/services/article-service"
  ARTICLE_STORAGE_DRIVER=postgres \
  ARTICLE_POSTGRES_DSN='postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable' \
  ARTICLE_CONSUMER_MAX_MESSAGES=1 \
  KAFKA_BROKERS=localhost:9092 \
  go run ./cmd/article-service
) >"$ARTICLE_LOG" 2>&1 &
ARTICLE_PID=$!

sleep 3

echo "Publishing sample article.discovered.v1 event..."
(
  cd "$ROOT_DIR/services/parser-service"
  KAFKA_BROKERS=localhost:9092 go run ./cmd/publish-discovered
)

echo "Waiting for article-service to consume..."
for _ in {1..30}; do
  if ! kill -0 "$ARTICLE_PID" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if kill -0 "$ARTICLE_PID" >/dev/null 2>&1; then
  echo "article-service did not stop after consuming one message"
  cat "$ARTICLE_LOG"
  exit 1
fi

ARTICLE_COUNT="$(
  docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U articleflow -d articleflow -tAc \
    "SELECT count(*) FROM articles WHERE external_id = 'dev-habr-article-1';"
)"

if [[ "$ARTICLE_COUNT" != "1" ]]; then
  echo "expected one stored article, got $ARTICLE_COUNT"
  cat "$ARTICLE_LOG"
  exit 1
fi

echo "E2E article chain passed: Kafka -> article-service -> Postgres"

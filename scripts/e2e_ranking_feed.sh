#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

source "$PROJECT_ROOT/scripts/env.sh" >/dev/null

RUN_ID="ranking-feed-$(date +%s)"
FEED_LOG="/tmp/articleflow-feed-${RUN_ID}.log"
RANKING_LOG="/tmp/articleflow-ranking-${RUN_ID}.log"
FEED_RESPONSE="/tmp/articleflow-feed-${RUN_ID}.json"

cleanup() {
  if [[ -n "${FEED_PID:-}" ]]; then
    kill "$FEED_PID" 2>/dev/null || true
    wait "$FEED_PID" 2>/dev/null || true
  fi
  if [[ -n "${RANKING_PID:-}" ]]; then
    kill "$RANKING_PID" 2>/dev/null || true
    wait "$RANKING_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

docker compose -f deployments/docker-compose.yml up -d kafka zookeeper postgres >/dev/null
for _ in {1..30}; do
  if docker compose -f deployments/docker-compose.yml exec -T postgres pg_isready -U articleflow -d articleflow >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
for migration in deployments/postgres/migrations/*.sql; do
  docker compose -f deployments/docker-compose.yml exec -T postgres psql -U articleflow -d articleflow <"$migration" >/dev/null
done
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic article.discovered.v1 --partitions 1 --replication-factor 1 >/dev/null
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic feed.item.scored.v1 --partitions 1 --replication-factor 1 >/dev/null
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic user.reaction.created.v1 --partitions 1 --replication-factor 1 >/dev/null

(
  cd services/feed-service
  KAFKA_BROKERS=127.0.0.1:9092 \
    FEED_HTTP_ADDR=:18082 \
    FEED_STORAGE_DRIVER=postgres \
    FEED_POSTGRES_DSN='postgres://articleflow:articleflow@localhost:5432/articleflow?sslmode=disable' \
    FEED_CONSUMER_GROUP_ID="feed-${RUN_ID}" \
    go run ./cmd/feed-service
) >"$FEED_LOG" 2>&1 &
FEED_PID=$!

(
  cd services/ranking-service
  KAFKA_BROKERS=127.0.0.1:9092 \
    RANKING_CONSUMER_GROUP_ID="ranking-${RUN_ID}" \
    go run ./cmd/ranking-service
) >"$RANKING_LOG" 2>&1 &
RANKING_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://localhost:18082/api/v1/feed?limit=1" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

sleep 2
(
  cd services/parser-service
  KAFKA_BROKERS=127.0.0.1:9092 go run ./cmd/publish-discovered
) >/tmp/articleflow-publish-"${RUN_ID}".log 2>&1

for _ in {1..30}; do
  curl -fsS "http://localhost:18082/api/v1/feed?limit=5" >"$FEED_RESPONSE"
  if rg -q "ArticleID" "$FEED_RESPONSE"; then
    echo "ranking/feed e2e passed"
    cat "$FEED_RESPONSE"
    exit 0
  fi
  sleep 1
done

echo "ranking/feed e2e failed"
echo "feed-service log:"
sed -n '1,160p' "$FEED_LOG"
echo "ranking-service log:"
sed -n '1,160p' "$RANKING_LOG"
echo "feed response:"
cat "$FEED_RESPONSE"
exit 1

#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

source "$PROJECT_ROOT/scripts/env.sh" >/dev/null

LOG_DIR="${ARTICLEFLOW_DEV_LOG_DIR:-/tmp/articleflow-dev}"
PID_FILE="$LOG_DIR/pids"
BIN_DIR="$LOG_DIR/bin"
mkdir -p "$LOG_DIR" "$BIN_DIR"

if [[ -f "$PID_FILE" ]]; then
  while read -r pid; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "dev stack already running; run make dev-stop first"
      exit 1
    fi
  done <"$PID_FILE"
fi
: >"$PID_FILE"

docker compose -f deployments/docker-compose.yml up -d kafka zookeeper >/dev/null
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic article.discovered.v1 --partitions 1 --replication-factor 1 >/dev/null
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic feed.item.scored.v1 --partitions 1 --replication-factor 1 >/dev/null
docker exec deployments-kafka-1 kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic user.reaction.created.v1 --partitions 1 --replication-factor 1 >/dev/null

start_service() {
  local name="$1"
  local workdir="$2"
  shift
  shift
  local command="$*"
  local pid
  pid="$(python3 "$PROJECT_ROOT/scripts/dev_spawn.py" --workdir "$workdir" --log "$LOG_DIR/${name}.log" -- "$command")"
  echo "$pid" >>"$PID_FILE"
  echo "$name pid=$pid log=$LOG_DIR/${name}.log"
}

build_service() {
  local name="$1"
  echo "building $name"
  (
    cd "$PROJECT_ROOT/services/$name"
    go build -o "$BIN_DIR/$name" "./cmd/$name"
  )
}

build_service parser-service
build_service ranking-service
build_service feed-service
build_service gateway-api

start_service parser-service "$PROJECT_ROOT/services/parser-service" env \
  KAFKA_BROKERS=localhost:9092 \
  PARSER_HTTP_ADDR=:8081 \
  HABR_REQUEST_DELAY_MS=500 \
  "$BIN_DIR/parser-service"

start_service ranking-service "$PROJECT_ROOT/services/ranking-service" env \
  KAFKA_BROKERS=localhost:9092 \
  RANKING_CONSUMER_GROUP_ID=ranking-service-dev \
  "$BIN_DIR/ranking-service"

start_service feed-service "$PROJECT_ROOT/services/feed-service" env \
  KAFKA_BROKERS=localhost:9092 \
  FEED_HTTP_ADDR=:8082 \
  FEED_CONSUMER_GROUP_ID=feed-service-dev \
  "$BIN_DIR/feed-service"

start_service gateway-api "$PROJECT_ROOT/services/gateway-api" env \
  GATEWAY_HTTP_ADDR=:8080 \
  PARSER_SERVICE_URL=http://localhost:8081 \
  FEED_SERVICE_URL=http://localhost:8082 \
  "$BIN_DIR/gateway-api"

start_service web "$PROJECT_ROOT" env \
  python3 -m http.server 5173 --bind 127.0.0.1 --directory "$PROJECT_ROOT/web"

wait_for() {
  local name="$1"
  local url="$2"
  for _ in {1..30}; do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      echo "$name ready"
      return 0
    fi
    sleep 1
  done
  echo "$name failed readiness: $url"
  echo "logs:"
  for log_file in "$LOG_DIR"/*.log; do
    echo "--- $log_file"
    sed -n '1,120p' "$log_file"
  done
  exit 1
}

wait_for parser-service "http://localhost:8081/healthz"
wait_for feed-service "http://localhost:8082/healthz"
wait_for gateway-api "http://localhost:8080/healthz"
wait_for web "http://127.0.0.1:5173/"

echo "dev stack started"
echo "web:     http://127.0.0.1:5173"
echo "gateway: http://localhost:8080"
echo "logs:    $LOG_DIR"

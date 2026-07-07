#!/usr/bin/env bash
set -euo pipefail

LOG_DIR="${ARTICLEFLOW_DEV_LOG_DIR:-/tmp/articleflow-dev}"
PID_FILE="$LOG_DIR/pids"

check_endpoint() {
  local name="$1"
  local url="$2"
  if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
    echo "$name: ok ($url)"
  else
    echo "$name: down ($url)"
  fi
}

echo "articleflow dev status"
echo "logs: $LOG_DIR"

if [[ -f "$PID_FILE" ]]; then
  echo "pids:"
  while read -r pid; do
    [[ -z "$pid" ]] && continue
    if kill -0 "$pid" 2>/dev/null; then
      echo "  $pid running"
    else
      echo "  $pid stopped"
    fi
  done <"$PID_FILE"
else
  echo "pids: none"
fi

echo "endpoints:"
check_endpoint "gateway" "http://localhost:8080/healthz"
check_endpoint "article" "http://localhost:8083/healthz"
check_endpoint "parser" "http://localhost:8081/healthz"
check_endpoint "feed" "http://localhost:8082/healthz"
check_endpoint "web" "http://127.0.0.1:5173/"

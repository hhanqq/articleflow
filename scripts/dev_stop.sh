#!/usr/bin/env bash
set -euo pipefail

LOG_DIR="${ARTICLEFLOW_DEV_LOG_DIR:-/tmp/articleflow-dev}"
PID_FILE="$LOG_DIR/pids"

if [[ ! -f "$PID_FILE" ]]; then
  echo "dev stack pid file not found"
  exit 0
fi

while read -r pid; do
  if [[ -n "$pid" ]]; then
    kill "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
  fi
done <"$PID_FILE"

while read -r pid; do
  if [[ -n "$pid" ]]; then
    wait "$pid" 2>/dev/null || true
  fi
done <"$PID_FILE"

for port in 8080 8081 8082 5173; do
  while read -r pid; do
    if [[ -n "$pid" ]]; then
      kill "$pid" 2>/dev/null || true
    fi
  done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
done

rm -f "$PID_FILE"
echo "dev stack stopped"

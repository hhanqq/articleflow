#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
QUERY="${QUERY:-go kafka}"
REQUESTS="${REQUESTS:-20}"
LIMIT="${LIMIT:-10}"
COOKIE_JAR="$(mktemp)"
trap 'rm -f "${COOKIE_JAR}"' EXIT

total_ms=0
for index in $(seq 1 "${REQUESTS}"); do
  started_ms="$(node -e 'console.log(Date.now())')"
  encoded_query="$(node -e 'console.log(encodeURIComponent(process.argv[1]))' "${QUERY}")"
  payload="$(curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" "${BASE_URL}/api/v1/feed?query=${encoded_query}&limit=${LIMIT}&refill=true")"
  finished_ms="$(node -e 'console.log(Date.now())')"
  elapsed_ms="$(( finished_ms - started_ms ))"
  total_ms="$(( total_ms + elapsed_ms ))"
  node - "${payload}" <<'NODE'
const payload = JSON.parse(process.argv[2]);
if (!Array.isArray(payload.items || payload.Items)) {
  console.error("feed response does not contain items array");
  process.exit(1);
}
NODE
  printf 'request=%s elapsed_ms=%s\n' "${index}" "${elapsed_ms}"
done

average_ms="$(( total_ms / REQUESTS ))"
printf 'load feed smoke ok: requests=%s average_ms=%s\n' "${REQUESTS}" "${average_ms}"

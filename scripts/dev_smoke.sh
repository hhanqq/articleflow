#!/usr/bin/env bash
set -euo pipefail

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
  echo "$name is not ready: $url"
  return 1
}

wait_for gateway "http://localhost:8080/healthz"
wait_for article "http://localhost:8083/healthz"
wait_for parser "http://localhost:8081/healthz"
wait_for feed "http://localhost:8082/healthz"
wait_for web "http://127.0.0.1:5173/"

curl -isS -X OPTIONS "http://localhost:8080/api/v1/search/jobs" \
  -H "Origin: http://127.0.0.1:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: content-type" >/tmp/articleflow-dev-cors.txt

if ! rg -q "204 No Content|Access-Control-Allow-Origin" /tmp/articleflow-dev-cors.txt; then
  echo "CORS preflight did not return expected headers"
  cat /tmp/articleflow-dev-cors.txt
  exit 1
fi

curl -fsS -X POST "http://localhost:8080/api/v1/search/jobs" \
  -H "Origin: http://127.0.0.1:5173" \
  -H "Content-Type: application/json" \
  -d '{"query":"go kafka","sources":["habr"],"limit":1}' >/tmp/articleflow-dev-job.json

if ! rg -q "parser-job-" /tmp/articleflow-dev-job.json; then
  echo "parser job was not created"
  cat /tmp/articleflow-dev-job.json
  exit 1
fi

job_id="$(sed -n 's/.*"ID":"\([^"]*\)".*/\1/p' /tmp/articleflow-dev-job.json | head -n 1)"
for _ in {1..45}; do
  curl -fsS "http://localhost:8080/api/v1/search/jobs/${job_id}" >/tmp/articleflow-dev-job-status.json
  if rg -q '"Status":"(completed|failed)"' /tmp/articleflow-dev-job-status.json; then
    break
  fi
  sleep 1
done

if ! rg -q '"Status":"completed"' /tmp/articleflow-dev-job-status.json; then
  echo "parser job did not complete"
  cat /tmp/articleflow-dev-job-status.json
  exit 1
fi

curl -fsS "http://localhost:8080/api/v1/feed?limit=3" >/tmp/articleflow-dev-feed.json
if ! rg -q '"items"' /tmp/articleflow-dev-feed.json; then
  echo "feed response is invalid"
  cat /tmp/articleflow-dev-feed.json
  exit 1
fi

node -e 'const fs=require("fs"); const payload=JSON.parse(fs.readFileSync("/tmp/articleflow-dev-feed.json","utf8")); for (const item of (payload.items||payload.Items||[])) { const id=String(item.ArticleID||item.article_id||item.id||""); if (id) console.log(id); }' >/tmp/articleflow-dev-feed-ids.txt
if [[ ! -s /tmp/articleflow-dev-feed-ids.txt ]]; then
  echo "feed did not include an article id"
  cat /tmp/articleflow-dev-feed.json
  exit 1
fi

article_found=0
while IFS= read -r article_id; do
  if curl -fsS -G "http://localhost:8080/api/v1/articles" \
    --data-urlencode "id=$article_id" >/tmp/articleflow-dev-article.json 2>/dev/null; then
    article_found=1
    break
  fi
done </tmp/articleflow-dev-feed-ids.txt
if [[ "$article_found" -ne 1 ]]; then
  echo "none of the feed articles had a persisted article detail"
  cat /tmp/articleflow-dev-feed.json
  exit 1
fi
if ! rg -q '"article"' /tmp/articleflow-dev-article.json; then
  echo "article detail response is invalid"
  cat /tmp/articleflow-dev-article.json
  exit 1
fi

curl -fsS -X POST "http://localhost:8080/api/v1/reactions" \
  -H "Origin: http://127.0.0.1:5173" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"reader-smoke","article_id":"habr:dev-habr-article-1","type":"save"}' >/tmp/articleflow-dev-reaction.json

if ! rg -q '"accepted":true' /tmp/articleflow-dev-reaction.json; then
  echo "reaction request was not accepted"
  cat /tmp/articleflow-dev-reaction.json
  exit 1
fi

for _ in {1..20}; do
  reaction_count="$(docker compose -f deployments/docker-compose.yml exec -T postgres psql -U articleflow -d articleflow -tAc "select count(*) from user_reactions where user_id = 'reader-smoke' and article_id = 'habr:dev-habr-article-1' and type = 'save';" | tr -d '[:space:]')"
  if [[ "$reaction_count" == "1" ]]; then
    break
  fi
  sleep 1
done

if [[ "${reaction_count:-0}" != "1" ]]; then
  echo "reaction was not persisted to Postgres"
  cat /tmp/articleflow-dev-reaction.json
  exit 1
fi

echo "dev smoke passed"

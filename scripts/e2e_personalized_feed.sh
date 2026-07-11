#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
USER_BASE_URL="${USER_BASE_URL:-http://localhost:8084}"
COOKIE_JAR="$(mktemp)"
trap 'rm -f "${COOKIE_JAR}"' EXIT

require_json_field() {
  node - "$1" "$2" <<'NODE'
const payload = JSON.parse(process.argv[2]);
const expr = process.argv[3];
if (expr === "first-id") {
  const item = (payload.items || payload.Items || [])[0];
  if (!item) process.exit(2);
  console.log(item.ArticleID || item.article_id || item.id);
}
if (expr === "second-id") {
  const item = (payload.items || payload.Items || [])[1];
  if (!item) process.exit(2);
  console.log(item.ArticleID || item.article_id || item.id);
}
NODE
}

me_payload="$(curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" "${BASE_URL}/api/v1/me")"
USER_ID="$(node - "${me_payload}" <<'NODE'
const payload = JSON.parse(process.argv[2]);
const user = payload.user || payload.User || {};
if (!user.ID && !user.id) process.exit(2);
console.log(user.ID || user.id);
NODE
)"

feed_payload="$(curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" "${BASE_URL}/api/v1/feed?limit=5")"
skip_article_id="$(require_json_field "${feed_payload}" "first-id")"
save_article_id="$(require_json_field "${feed_payload}" "second-id")"

curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" -X POST "${BASE_URL}/api/v1/reactions" \
  -H "Content-Type: application/json" \
  -d "{\"article_id\":\"${skip_article_id}\",\"type\":\"skip\"}" >/dev/null

curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" -X POST "${BASE_URL}/api/v1/reactions" \
  -H "Content-Type: application/json" \
  -d "{\"article_id\":\"${save_article_id}\",\"type\":\"save\"}" >/dev/null

reactions_seen=false
for _ in {1..10}; do
  reactions="$(curl -fsS "${USER_BASE_URL}/api/v1/users/${USER_ID}/reactions")"
  if [[ "${reactions}" == *"${skip_article_id}"* && "${reactions}" == *"${save_article_id}"* ]]; then
    reactions_seen=true
    break
  fi
  sleep 1
done

if [[ "${reactions_seen}" != "true" ]]; then
  echo "user-service did not expose both reactions for ${USER_ID}" >&2
  exit 1
fi

personalized="$(curl -fsS -c "${COOKIE_JAR}" -b "${COOKIE_JAR}" "${BASE_URL}/api/v1/feed?limit=5")"

node - "${personalized}" "${skip_article_id}" "${save_article_id}" <<'NODE'
const payload = JSON.parse(process.argv[2]);
const skipID = process.argv[3];
const saveID = process.argv[4];
const items = payload.items || payload.Items || [];
if (items.some((item) => (item.ArticleID || item.article_id || item.id) === skipID)) {
  console.error(`skipped article still present: ${skipID}`);
  process.exit(1);
}
const saved = items.find((item) => (item.ArticleID || item.article_id || item.id) === saveID);
if (!saved) {
  console.error(`saved article missing from personalized feed: ${saveID}`);
  process.exit(1);
}
if ((saved.Reaction || saved.reaction) !== "save" || !(saved.Saved || saved.saved)) {
  console.error(`saved marker missing for article: ${saveID}`);
  process.exit(1);
}
console.log(`personalized feed e2e ok: skip=${skipID} save=${saveID}`);
NODE

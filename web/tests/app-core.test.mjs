import test from "node:test";
import assert from "node:assert/strict";

import {
  buildReactionPayload,
  buildSearchJobPayload,
  formatScore,
  normalizeJobResponse,
  normalizeFeedItem,
  shouldPollJob,
} from "../src/app-core.mjs";

test("normalizeFeedItem accepts Go JSON field names from gateway", () => {
  const item = normalizeFeedItem({
    ArticleID: "habr:1",
    Title: "Go Kafka",
    Summary: "Streaming article",
    SourceName: "habr",
    URL: "https://habr.com/1",
    Tags: ["go", "kafka"],
    Score: 23.45,
    PublishedAt: "2026-07-07T12:00:00Z",
  });

  assert.equal(item.id, "habr:1");
  assert.equal(item.title, "Go Kafka");
  assert.equal(item.sourceName, "habr");
  assert.equal(item.url, "https://habr.com/1");
  assert.deepEqual(item.tags, ["go", "kafka"]);
  assert.equal(item.score, 23.45);
});

test("buildSearchJobPayload trims query and keeps selected sources", () => {
  const payload = buildSearchJobPayload({
    query: "  go kafka  ",
    sources: ["habr"],
    limit: "5",
  });

  assert.deepEqual(payload, {
    query: "go kafka",
    sources: ["habr"],
    limit: 5,
  });
});

test("buildReactionPayload validates article and reaction", () => {
  const payload = buildReactionPayload({
    userID: "reader-1",
    articleID: "habr:1",
    type: "save",
  });

  assert.deepEqual(payload, {
    user_id: "reader-1",
    article_id: "habr:1",
    type: "save",
  });
});

test("shouldPollJob only polls active job statuses", () => {
  assert.equal(shouldPollJob({ Status: "queued" }), true);
  assert.equal(shouldPollJob({ Status: "running" }), true);
  assert.equal(shouldPollJob({ Status: "completed" }), false);
  assert.equal(shouldPollJob({ Status: "failed" }), false);
});

test("normalizeJobResponse unwraps gateway job payload", () => {
  const job = normalizeJobResponse({
    job: {
      ID: "parser-job-1",
      Status: "queued",
      CandidatesCount: 0,
    },
  });

  assert.equal(job.ID, "parser-job-1");
  assert.equal(job.Status, "queued");
});

test("formatScore returns one decimal for finite scores", () => {
  assert.equal(formatScore(23.456), "23.5");
  assert.equal(formatScore(Number.NaN), "0.0");
});

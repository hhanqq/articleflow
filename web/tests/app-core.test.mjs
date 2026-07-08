import test from "node:test";
import assert from "node:assert/strict";

import {
  buildReactionPayload,
  buildSearchJobPayload,
  formatScore,
  normalizeArticle,
  normalizeCandidateItem,
  normalizeJobResponse,
  normalizeFeedItem,
  normalizeSearchResponse,
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

test("normalizeCandidateItem builds stable feed item id from parser candidate", () => {
  const item = normalizeCandidateItem({
    SourceName: "vc",
    ExternalID: "42",
    Title: "Путешествие в Китай",
    Summary: "Маршрут и бюджет",
    URL: "https://vc.ru/story/42",
    Tags: ["travel"],
  });

  assert.equal(item.id, "vc:42");
  assert.equal(item.title, "Путешествие в Китай");
  assert.equal(item.sourceName, "vc");
  assert.equal(item.url, "https://vc.ru/story/42");
  assert.deepEqual(item.tags, ["travel"]);
  assert.equal(item.score, 0);
});

test("normalizeArticle accepts article detail payload from gateway", () => {
  const article = normalizeArticle({
    article: {
      ID: "habr:1",
      Title: "Go Kafka",
      Summary: "Summary",
      Content: "<p>Full text</p>",
      Author: "author",
      Tags: ["go"],
      URL: "https://habr.com/1",
      PublishedAt: "2026-07-07T12:00:00Z",
    },
  });

  assert.equal(article.id, "habr:1");
  assert.equal(article.title, "Go Kafka");
  assert.equal(article.content, "<p>Full text</p>");
  assert.deepEqual(article.tags, ["go"]);
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

test("normalizeSearchResponse unwraps gateway candidates", () => {
  const candidates = normalizeSearchResponse({
    candidates: [{ Title: "Go Kafka" }],
  });

  assert.equal(candidates.length, 1);
  assert.equal(candidates[0].Title, "Go Kafka");
});

test("formatScore returns one decimal for finite scores", () => {
  assert.equal(formatScore(23.456), "23.5");
  assert.equal(formatScore(Number.NaN), "0.0");
});

import test from "node:test";
import assert from "node:assert/strict";

import {
  buildFeedQueryPath,
  buildFreshnessRange,
  buildReactionPayload,
  buildSearchJobPayload,
  buildSelectedSources,
  defaultAPIBase,
  formatDate,
  formatScore,
  normalizeArticle,
  normalizeCandidateItem,
  normalizeJobResponse,
  normalizeJobsResponse,
  normalizeFeedItem,
  normalizeSearchResponse,
  normalizeSourceStats,
  normalizeSourcesResponse,
  shouldPollJob,
} from "../src/app-core.mjs";

test("defaultAPIBase uses gateway in local dev and same origin behind nginx", () => {
  assert.equal(defaultAPIBase("http://127.0.0.1:5173"), "http://localhost:8080");
  assert.equal(defaultAPIBase("http://localhost:5173"), "http://localhost:8080");
  assert.equal(defaultAPIBase("http://localhost:8089"), "http://localhost:8089");
  assert.equal(defaultAPIBase("https://articles.example.com"), "https://articles.example.com");
});

test("formatDate hides zero Go time values", () => {
  assert.equal(formatDate("0001-01-01T00:00:00Z"), "");
});

test("normalizeFeedItem accepts Go JSON field names from gateway", () => {
  const item = normalizeFeedItem({
    ArticleID: "habr:1",
    Title: "Go Kafka",
    Summary: "Streaming article",
    SourceName: "habr",
    URL: "https://habr.com/1",
    Tags: ["go", "kafka"],
    Score: 23.45,
    ScoreReasons: ["query_match:go", "freshness"],
    PublishedAt: "2026-07-07T12:00:00Z",
  });

  assert.equal(item.id, "habr:1");
  assert.equal(item.title, "Go Kafka");
  assert.equal(item.sourceName, "habr");
  assert.equal(item.url, "https://habr.com/1");
  assert.deepEqual(item.tags, ["go", "kafka"]);
  assert.equal(item.score, 23.45);
  assert.deepEqual(item.scoreReasons, ["query_match:go", "freshness"]);
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

test("buildSearchJobPayload includes date filters when provided", () => {
  const payload = buildSearchJobPayload({
    query: "  новости  ",
    sources: ["dzen"],
    limit: "5",
    fromDate: "2026-07-01T00:00:00.000Z",
    toDate: "2026-07-11T00:00:00.000Z",
  });

  assert.equal(payload.from_date, "2026-07-01T00:00:00.000Z");
  assert.equal(payload.to_date, "2026-07-11T00:00:00.000Z");
});

test("buildSelectedSources uses empty list for all sources", () => {
  assert.deepEqual(buildSelectedSources({ allSelected: true, selected: ["habr", "vc"] }), []);
});

test("buildSelectedSources keeps explicit unique source list", () => {
  assert.deepEqual(
    buildSelectedSources({ allSelected: false, selected: ["habr", "vc", "habr", ""] }),
    ["habr", "vc"],
  );
});

test("buildFeedQueryPath builds cursor feed URL for query scrolling", () => {
  assert.equal(
    buildFeedQueryPath({
      query: "  go kafka  ",
      sources: ["habr", "vc"],
      limit: 10,
      cursor: "offset:20",
      refill: true,
      fromDate: "2026-07-01T00:00:00.000Z",
    }),
    "/api/v1/feed?limit=10&query=go+kafka&sources=habr%2Cvc&cursor=offset%3A20&from_date=2026-07-01T00%3A00%3A00.000Z&refill=true",
  );
});

test("buildFeedQueryPath omits empty optional params", () => {
  assert.equal(buildFeedQueryPath({ limit: "bad", sources: [], cursor: "" }), "/api/v1/feed?limit=30");
});

test("buildFreshnessRange maps presets to from_date values", () => {
  const now = new Date("2026-07-11T12:00:00.000Z");

  assert.deepEqual(buildFreshnessRange("all", now), {});
  assert.deepEqual(buildFreshnessRange("week", now), {
    fromDate: "2026-07-04T12:00:00.000Z",
  });
  assert.deepEqual(buildFreshnessRange("month", now), {
    fromDate: "2026-06-11T12:00:00.000Z",
  });
});

test("normalizeSourceStats accepts Go JSON field names", () => {
  const stat = normalizeSourceStats({
    SourceName: "vc",
    Strategy: "html",
    Status: "ok",
    FoundCount: 12,
    AcceptedCount: 8,
    ReturnedCount: 4,
    PublishedCount: 4,
    FilteredCount: 4,
    DurationMS: 123,
  });

  assert.deepEqual(stat, {
    sourceName: "vc",
    strategy: "html",
    status: "ok",
    foundCount: 12,
    acceptedCount: 8,
    returnedCount: 4,
    publishedCount: 4,
    filteredCount: 4,
    error: "",
    durationMS: 123,
  });
});

test("normalizeSourcesResponse accepts Go JSON parser sources", () => {
  const sources = normalizeSourcesResponse({
    sources: [
      { Name: "habr", DisplayName: "Habr", Kind: "html_rss", Enabled: true, Searchable: true },
      { Name: "vc_rss", DisplayName: "vc.ru RSS", Kind: "rss", Enabled: false, Searchable: true },
    ],
  });

  assert.deepEqual(sources, [
    { name: "habr", displayName: "Habr", kind: "html_rss", enabled: true, searchable: true, notes: "" },
    { name: "vc_rss", displayName: "vc.ru RSS", kind: "rss", enabled: false, searchable: true, notes: "" },
  ]);
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

test("normalizeJobsResponse unwraps gateway jobs history payload", () => {
  const jobs = normalizeJobsResponse({
    jobs: [
      { ID: "parser-job-2", Status: "completed", CandidatesCount: 3 },
      { ID: "parser-job-1", Status: "failed", Error: "vc unavailable" },
    ],
  });

  assert.equal(jobs.length, 2);
  assert.equal(jobs[0].ID, "parser-job-2");
  assert.equal(jobs[1].Error, "vc unavailable");
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

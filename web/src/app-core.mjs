const ACTIVE_JOB_STATUSES = new Set(["queued", "running"]);
const REACTION_TYPES = new Set(["like", "dislike", "skip", "save", "open"]);

export function defaultAPIBase(origin = "") {
  const normalized = String(origin || "").replace(/\/+$/, "");
  if (normalized === "http://127.0.0.1:5173" || normalized === "http://localhost:5173") {
    return "http://localhost:8080";
  }
  return normalized || "http://localhost:8080";
}

export function normalizeFeedItem(raw = {}) {
  const tags = raw.Tags ?? raw.tags ?? [];
  const scoreReasons = raw.ScoreReasons ?? raw.score_reasons ?? [];
  return {
    id: String(raw.ArticleID ?? raw.article_id ?? raw.id ?? ""),
    title: String(raw.Title ?? raw.title ?? "Untitled"),
    summary: String(raw.Summary ?? raw.summary ?? ""),
    sourceName: String(raw.SourceName ?? raw.source_name ?? ""),
    url: String(raw.URL ?? raw.url ?? ""),
    tags: Array.isArray(tags) ? tags : [],
    score: Number(raw.Score ?? raw.score ?? 0),
    scoreReasons: Array.isArray(scoreReasons) ? scoreReasons : [],
    reaction: String(raw.Reaction ?? raw.reaction ?? ""),
    saved: Boolean(raw.Saved ?? raw.saved),
    publishedAt: String(raw.PublishedAt ?? raw.published_at ?? ""),
  };
}

export function normalizeCandidateItem(raw = {}) {
  const sourceName = String(raw.SourceName ?? raw.source_name ?? "");
  const externalID = String(raw.ExternalID ?? raw.external_id ?? "");
  const fallbackID = sourceName && externalID ? `${sourceName}:${externalID}` : raw.URL ?? raw.url ?? "";
  return normalizeFeedItem({
    ...raw,
    ArticleID: raw.ArticleID ?? raw.article_id ?? raw.ID ?? raw.id ?? fallbackID,
    SourceName: raw.SourceName ?? raw.source_name ?? sourceName,
    Score: raw.Score ?? raw.score ?? 0,
  });
}

export function normalizeArticle(raw = {}) {
  const payload = raw.article ?? raw.Article ?? raw;
  const tags = payload.Tags ?? payload.tags ?? [];
  return {
    id: String(payload.ID ?? payload.id ?? ""),
    title: String(payload.Title ?? payload.title ?? "Untitled"),
    summary: String(payload.Summary ?? payload.summary ?? ""),
    content: String(payload.Content ?? payload.content ?? ""),
    author: String(payload.Author ?? payload.author ?? ""),
    sourceName: String(payload.SourceName ?? payload.source_name ?? ""),
    externalID: String(payload.ExternalID ?? payload.external_id ?? ""),
    url: String(payload.URL ?? payload.url ?? ""),
    tags: Array.isArray(tags) ? tags : [],
    language: String(payload.Language ?? payload.language ?? ""),
    publishedAt: String(payload.PublishedAt ?? payload.published_at ?? ""),
    parsedAt: String(payload.ParsedAt ?? payload.parsed_at ?? ""),
  };
}

export function buildSearchJobPayload({ query, sources, limit, fromDate, toDate }) {
  const normalizedQuery = String(query ?? "").trim();
  if (!normalizedQuery) {
    throw new Error("query is required");
  }
  const parsedLimit = Number.parseInt(String(limit ?? "5"), 10);
  const payload = {
    query: normalizedQuery,
    sources: Array.isArray(sources) ? sources.filter(Boolean) : [],
    limit: Number.isFinite(parsedLimit) && parsedLimit > 0 ? parsedLimit : 5,
  };
  const normalizedFromDate = String(fromDate ?? "").trim();
  const normalizedToDate = String(toDate ?? "").trim();
  if (normalizedFromDate) {
    payload.from_date = normalizedFromDate;
  }
  if (normalizedToDate) {
    payload.to_date = normalizedToDate;
  }
  return payload;
}

export function buildSelectedSources({ allSelected, selected }) {
  if (allSelected) {
    return [];
  }
  const unique = [];
  const seen = new Set();
  for (const source of Array.isArray(selected) ? selected : []) {
    const normalized = String(source ?? "").trim();
    if (!normalized || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    unique.push(normalized);
  }
  return unique;
}

export function buildFeedQueryPath({ query, sources, limit, cursor, fromDate, toDate, refill, userID } = {}) {
  const params = new URLSearchParams();
  const parsedLimit = Number.parseInt(String(limit ?? "30"), 10);
  params.set("limit", String(Number.isFinite(parsedLimit) && parsedLimit > 0 ? parsedLimit : 30));

  const normalizedUserID = String(userID ?? "").trim();
  if (normalizedUserID) {
    params.set("user_id", normalizedUserID);
  }
  const normalizedQuery = String(query ?? "").trim();
  if (normalizedQuery) {
    params.set("query", normalizedQuery);
  }
  const selectedSources = Array.isArray(sources) ? sources.filter(Boolean) : [];
  if (selectedSources.length > 0) {
    params.set("sources", selectedSources.join(","));
  }
  const normalizedCursor = String(cursor ?? "").trim();
  if (normalizedCursor) {
    params.set("cursor", normalizedCursor);
  }
  const normalizedFromDate = String(fromDate ?? "").trim();
  if (normalizedFromDate) {
    params.set("from_date", normalizedFromDate);
  }
  const normalizedToDate = String(toDate ?? "").trim();
  if (normalizedToDate) {
    params.set("to_date", normalizedToDate);
  }
  if (refill) {
    params.set("refill", "true");
  }
  return `/api/v1/feed?${params.toString()}`;
}

export function buildFreshnessRange(value, now = new Date()) {
  const daysByValue = new Map([
    ["day", 1],
    ["week", 7],
    ["month", 30],
    ["year", 365],
  ]);
  const days = daysByValue.get(String(value ?? "").trim());
  if (!days) {
    return {};
  }
  const from = new Date(now.getTime() - days * 24 * 60 * 60 * 1000);
  return { fromDate: from.toISOString() };
}

export function applyLocalFeedReaction(items, articleID, type) {
  const normalizedID = String(articleID ?? "").trim();
  const normalizedType = String(type ?? "").trim();
  const list = Array.isArray(items) ? items : [];
  if (!normalizedID || !REACTION_TYPES.has(normalizedType)) {
    return list;
  }
  if (normalizedType === "skip") {
    return list.filter((item) => (item.id || item.url) !== normalizedID);
  }
  return list.map((item) => {
    if ((item.id || item.url) !== normalizedID) {
      return item;
    }
    return {
      ...item,
      reaction: normalizedType,
      saved: normalizedType === "save" ? true : item.saved,
    };
  });
}

export function buildFeedStatusParts({
  query,
  sources,
  fromDate,
  nextCursor,
  refillStarted,
  refillJobID,
  formatDateValue = formatDate,
} = {}) {
  const parts = [];
  const normalizedQuery = String(query ?? "").trim();
  if (normalizedQuery) {
    parts.push(`query: ${normalizedQuery}`);
  }
  const selectedSources = Array.isArray(sources) ? sources.filter(Boolean) : [];
  if (selectedSources.length > 0) {
    parts.push(`sources: ${selectedSources.join(", ")}`);
  }
  const normalizedFromDate = String(fromDate ?? "").trim();
  if (normalizedFromDate) {
    parts.push(`from: ${formatDateValue(normalizedFromDate) || normalizedFromDate}`);
  }
  const normalizedCursor = String(nextCursor ?? "").trim();
  if (normalizedCursor) {
    parts.push(`next: ${normalizedCursor}`);
  }
  if (refillJobID) {
    parts.push(`refill: ${refillJobID}`);
  } else if (refillStarted) {
    parts.push("refill started");
  }
  return parts;
}

export function normalizeSourceStats(raw = {}) {
  return {
    sourceName: String(raw.SourceName ?? raw.source_name ?? ""),
    strategy: String(raw.Strategy ?? raw.strategy ?? ""),
    status: String(raw.Status ?? raw.status ?? ""),
    foundCount: Number(raw.FoundCount ?? raw.found_count ?? 0),
    acceptedCount: Number(raw.AcceptedCount ?? raw.accepted_count ?? 0),
    returnedCount: Number(raw.ReturnedCount ?? raw.returned_count ?? 0),
    publishedCount: Number(raw.PublishedCount ?? raw.published_count ?? 0),
    filteredCount: Number(raw.FilteredCount ?? raw.filtered_count ?? 0),
    error: String(raw.Error ?? raw.error ?? ""),
    durationMS: Number(raw.DurationMS ?? raw.duration_ms ?? 0),
  };
}

export function normalizeSourcesResponse(raw = {}) {
  const sources = raw.sources ?? raw.Sources ?? [];
  if (!Array.isArray(sources)) {
    return [];
  }
  return sources.map((source) => ({
    name: String(source.Name ?? source.name ?? ""),
    displayName: String(source.DisplayName ?? source.display_name ?? source.Name ?? source.name ?? ""),
    kind: String(source.Kind ?? source.kind ?? ""),
    enabled: Boolean(source.Enabled ?? source.enabled),
    searchable: Boolean(source.Searchable ?? source.searchable),
    notes: String(source.Notes ?? source.notes ?? ""),
  })).filter((source) => source.name);
}

export function buildReactionPayload({ userID, articleID, type }) {
  const normalizedUserID = String(userID ?? "").trim();
  const normalizedArticleID = String(articleID ?? "").trim();
  const normalizedType = String(type ?? "").trim();
  if (!normalizedUserID) {
    throw new Error("user_id is required");
  }
  if (!normalizedArticleID) {
    throw new Error("article_id is required");
  }
  if (!REACTION_TYPES.has(normalizedType)) {
    throw new Error("unsupported reaction type");
  }
  return {
    user_id: normalizedUserID,
    article_id: normalizedArticleID,
    type: normalizedType,
  };
}

export function shouldPollJob(job) {
  const status = String(job?.Status ?? job?.status ?? "").toLowerCase();
  return ACTIVE_JOB_STATUSES.has(status);
}

export function normalizeJobResponse(raw = {}) {
  return raw.job ?? raw.Job ?? raw;
}

export function normalizeJobsResponse(raw = {}) {
  const jobs = raw.jobs ?? raw.Jobs ?? [];
  return Array.isArray(jobs) ? jobs : [];
}

export function normalizeSearchResponse(raw = {}) {
  const candidates = raw.candidates ?? raw.Candidates ?? [];
  return Array.isArray(candidates) ? candidates : [];
}

export function formatScore(score) {
  const value = Number(score);
  if (!Number.isFinite(value)) {
    return "0.0";
  }
  return value.toFixed(1);
}

export function formatDate(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  if (date.getUTCFullYear() < 2000) {
    return "";
  }
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

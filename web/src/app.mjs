import {
  applyLocalFeedReaction,
  buildFeedQueryPath,
  buildFeedStatusParts,
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
} from "./app-core.mjs";

const state = {
  apiBase: localStorage.getItem("articleflow.apiBase") || defaultAPIBase(window.location.origin),
  items: [],
  selectedIndex: 0,
  activeJob: null,
  jobHistory: [],
  parserSources: [],
  selectedArticle: null,
  articleLoadingID: "",
  feedQuery: "",
  feedSources: [],
  feedLimit: 30,
  feedFromDate: "",
  feedToDate: "",
  nextCursor: "",
  feedLoadingMore: false,
  feedHasMore: false,
  pollTimer: 0,
  feedEmptyTitle: "",
  feedEmptyMessage: "",
};

const elements = {
  apiBase: document.querySelector("#apiBase"),
  feed: document.querySelector("#feed"),
  feedCount: document.querySelector("#feedCount"),
  refreshFeed: document.querySelector("#refreshFeed"),
  loadMoreFeed: document.querySelector("#loadMoreFeed"),
  feedPageStatus: document.querySelector("#feedPageStatus"),
  searchForm: document.querySelector("#searchForm"),
  searchStored: document.querySelector("#searchStored"),
  runParserJob: document.querySelector("#runParserJob"),
  searchQuery: document.querySelector("#searchQuery"),
  searchLimit: document.querySelector("#searchLimit"),
  searchFreshness: document.querySelector("#searchFreshness"),
  jobStatus: document.querySelector("#jobStatus"),
  jobMeta: document.querySelector("#jobMeta"),
  refreshJobs: document.querySelector("#refreshJobs"),
  jobHistory: document.querySelector("#jobHistory"),
  sourceAll: document.querySelector("#sourceAll"),
  sourceControls: document.querySelector("#sourceControls"),
  sourceStats: document.querySelector("#sourceStats"),
  lastError: document.querySelector("#lastError"),
  articleDetail: document.querySelector("#articleDetail"),
};

elements.apiBase.value = state.apiBase;

elements.apiBase.addEventListener("change", () => {
  state.apiBase = trimTrailingSlash(elements.apiBase.value);
  elements.apiBase.value = state.apiBase;
  localStorage.setItem("articleflow.apiBase", state.apiBase);
});

elements.refreshFeed.addEventListener("click", () => {
  void loadFeed();
});

elements.loadMoreFeed.addEventListener("click", () => {
  void loadMoreFeed();
});

elements.refreshJobs.addEventListener("click", () => {
  void loadJobHistory();
});

elements.searchForm.addEventListener("submit", (event) => {
  event.preventDefault();
  void searchStoredArticles();
});

elements.runParserJob.addEventListener("click", () => {
  void startSearchJob();
});

elements.sourceAll.addEventListener("change", () => {
  syncSourceControls();
});

elements.sourceControls.addEventListener("change", (event) => {
  const option = event.target.closest("[data-source-option]");
  if (option) {
    if (option.checked) {
      elements.sourceAll.checked = false;
    }
    syncSourceControls();
  }
});

elements.feed.addEventListener("click", (event) => {
  const detailButton = event.target.closest("[data-open-detail]");
  if (detailButton) {
    const articleID = detailButton.closest("[data-article-id]")?.dataset.articleId;
    void openArticleDetail(articleID);
    return;
  }

  const button = event.target.closest("[data-reaction]");
  if (!button) {
    return;
  }
  const articleID = button.closest("[data-article-id]")?.dataset.articleId;
  void sendReaction(articleID, button.dataset.reaction);
});

elements.feed.addEventListener("scroll", () => {
  if (!state.feedHasMore || state.feedLoadingMore || !state.nextCursor) {
    return;
  }
  const remaining = elements.feed.scrollHeight - elements.feed.scrollTop - elements.feed.clientHeight;
  if (remaining < 240) {
    void loadMoreFeed();
  }
});

elements.articleDetail.addEventListener("click", (event) => {
  if (event.target.closest("[data-close-detail]")) {
    state.selectedArticle = null;
    renderArticleDetail();
  }
});

elements.jobHistory.addEventListener("click", (event) => {
  const button = event.target.closest("[data-job-id]");
  if (!button) {
    return;
  }
  void openJob(button.dataset.jobId);
});

void loadFeed();
void loadJobHistory();
void loadSources();
void loadCurrentUser();

async function loadCurrentUser() {
  try {
    await requestJSON("/api/v1/me");
  } catch (error) {
    setError(error.message);
  }
}

async function loadFeed() {
  setError("");
  elements.refreshFeed.disabled = true;
  try {
    resetFeedPaging();
    const payload = await requestJSON(buildFeedQueryPath({ limit: 30 }));
    state.items = (payload.items || payload.Items || []).map(normalizeFeedItem);
    state.nextCursor = "";
    state.feedHasMore = false;
    state.feedEmptyTitle = "Лента пока пустая";
    state.feedEmptyMessage = "Запусти поиск справа или проверь, что gateway-api и feed-service доступны.";
    renderFeed();
  } catch (error) {
    setError(error.message);
  } finally {
    elements.refreshFeed.disabled = false;
    renderFeedPaging();
  }
}

async function startSearchJob() {
  setError("");
  clearTimeout(state.pollTimer);
  elements.runParserJob.disabled = true;
  try {
    const searchPayload = buildSearchJobPayload({
      query: elements.searchQuery.value,
      sources: selectedSources(),
      limit: elements.searchLimit.value,
      ...selectedFreshnessRange(),
    });
    const jobPayload = await requestJSON("/api/v1/search/jobs", {
      method: "POST",
      body: JSON.stringify(searchPayload),
    });
    state.activeJob = normalizeJobResponse(jobPayload);
    renderJob(state.activeJob);
    void loadJobHistory();
    scheduleJobPoll();
  } catch (error) {
    setError(error.message);
  } finally {
    elements.runParserJob.disabled = false;
  }
}

async function searchStoredArticles() {
  setError("");
  elements.searchStored.disabled = true;
  try {
    const searchPayload = buildSearchJobPayload({
      query: elements.searchQuery.value,
      sources: selectedSources(),
      limit: elements.searchLimit.value,
      ...selectedFreshnessRange(),
    });
    state.feedQuery = searchPayload.query;
    state.feedSources = searchPayload.sources;
    state.feedLimit = searchPayload.limit;
    state.feedFromDate = searchPayload.from_date || "";
    state.feedToDate = searchPayload.to_date || "";
    state.nextCursor = "";
    state.feedHasMore = false;
    const payload = await requestJSON(buildFeedQueryPath({
      query: state.feedQuery,
      sources: state.feedSources,
      limit: state.feedLimit,
      fromDate: state.feedFromDate,
      toDate: state.feedToDate,
      refill: true,
    }));
    state.items = (payload.items || payload.Items || []).map(normalizeFeedItem);
    state.nextCursor = payload.next_cursor || payload.NextCursor || "";
    state.feedHasMore = Boolean(state.nextCursor);
    if (state.items.length === 0) {
      state.feedEmptyTitle = "В базе ничего не найдено";
      state.feedEmptyMessage = `По запросу "${searchPayload.query}" нет сохраненных статей. Запусти parser job, чтобы попробовать подтянуть свежие материалы.`;
    } else {
      state.feedEmptyTitle = "";
      state.feedEmptyMessage = "";
    }
    renderFeed();
    renderFeedPaging(payload);
  } catch (error) {
    setError(error.message);
  } finally {
    elements.searchStored.disabled = false;
  }
}

async function loadMoreFeed() {
  if (!state.nextCursor || state.feedLoadingMore || !state.feedQuery) {
    return;
  }
  state.feedLoadingMore = true;
  renderFeedPaging();
  try {
    const payload = await requestJSON(buildFeedQueryPath({
      query: state.feedQuery,
      sources: state.feedSources,
      limit: state.feedLimit,
      cursor: state.nextCursor,
      fromDate: state.feedFromDate,
      toDate: state.feedToDate,
      refill: true,
    }));
    const nextItems = (payload.items || payload.Items || []).map(normalizeFeedItem);
    appendUniqueFeedItems(nextItems);
    state.nextCursor = payload.next_cursor || payload.NextCursor || "";
    state.feedHasMore = Boolean(state.nextCursor);
    renderFeed();
    renderFeedPaging(payload);
  } catch (error) {
    setError(error.message);
  } finally {
    state.feedLoadingMore = false;
    renderFeedPaging();
  }
}

function scheduleJobPoll() {
  clearTimeout(state.pollTimer);
  if (!shouldPollJob(state.activeJob)) {
    if (String(state.activeJob?.Status ?? state.activeJob?.status ?? "") === "completed") {
      renderCompletedJobResults(state.activeJob);
      void loadJobHistory();
    }
    return;
  }
  state.pollTimer = window.setTimeout(async () => {
    try {
      const jobID = state.activeJob.ID ?? state.activeJob.id;
      const payload = await requestJSON(`/api/v1/search/jobs/${encodeURIComponent(jobID)}`);
      state.activeJob = normalizeJobResponse(payload);
      renderJob(state.activeJob);
      if (!shouldPollJob(state.activeJob)) {
        void loadJobHistory();
      }
      scheduleJobPoll();
    } catch (error) {
      setError(error.message);
    }
  }, 1200);
}

async function loadJobHistory() {
  try {
    const payload = await requestJSON("/api/v1/search/jobs?limit=8");
    state.jobHistory = normalizeJobsResponse(payload);
    renderJobHistory();
  } catch (error) {
    state.jobHistory = [];
    renderJobHistory(error.message);
  }
}

async function loadSources() {
  try {
    const payload = await requestJSON("/api/v1/search/sources");
    state.parserSources = normalizeSourcesResponse(payload);
    renderSourceControls();
  } catch (error) {
    state.parserSources = fallbackSources();
    renderSourceControls();
  }
}

async function openJob(jobID) {
  const normalizedID = String(jobID ?? "").trim();
  if (!normalizedID) {
    return;
  }
  setError("");
  try {
    const payload = await requestJSON(`/api/v1/search/jobs/${encodeURIComponent(normalizedID)}`);
    state.activeJob = normalizeJobResponse(payload);
    renderJob(state.activeJob);
    if (!shouldPollJob(state.activeJob)) {
      renderCompletedJobResults(state.activeJob);
    } else {
      scheduleJobPoll();
    }
  } catch (error) {
    setError(error.message);
  }
}

function renderCompletedJobResults(job) {
  const candidates = job.Candidates ?? job.candidates ?? [];
  state.items = candidates.map(normalizeCandidateItem);
  state.nextCursor = "";
  state.feedHasMore = false;
  const query = job.Query?.Text ?? job.query?.text ?? elements.searchQuery.value.trim();
  if (state.items.length === 0) {
    state.feedEmptyTitle = "По запросу ничего не найдено";
    state.feedEmptyMessage = query
      ? `Parser job завершился: по запросу "${query}" найдено 0 статей в выбранных источниках.`
      : "Parser job завершился: найдено 0 статей в выбранных источниках.";
  } else {
    state.feedEmptyTitle = "";
    state.feedEmptyMessage = "";
  }
  renderFeed();
  renderFeedPaging();
}

async function sendReaction(articleID, type) {
  setError("");
  try {
    const payload = buildReactionPayload({ articleID, type });
    await requestJSON("/api/v1/reactions", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    state.items = applyLocalFeedReaction(state.items, articleID, type);
    if (type === "skip") {
      renderFeed();
      renderFeedPaging();
      return;
    }
    renderFeed();
    markReaction(articleID, type);
  } catch (error) {
    setError(error.message);
  }
}

async function openArticleDetail(articleID) {
  setError("");
  const normalizedID = String(articleID ?? "").trim();
  if (!normalizedID) {
    setError("article_id is required");
    return;
  }
  state.selectedArticle = null;
  state.articleLoadingID = normalizedID;
  renderArticleDetail();
  try {
    const payload = await requestJSON(`/api/v1/articles?id=${encodeURIComponent(normalizedID)}`);
    state.selectedArticle = normalizeArticle(payload);
    await sendReaction(normalizedID, "open");
  } catch (error) {
    setError(error.message);
  } finally {
    state.articleLoadingID = "";
    renderArticleDetail();
  }
}

async function requestJSON(path, options = {}) {
  let response;
  try {
    response = await fetch(`${state.apiBase}${path}`, {
      ...options,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(options.headers || {}),
      },
    });
  } catch (error) {
    throw new Error(`Не удалось подключиться к ${state.apiBase}. Проверь, что gateway-api запущен и API указан верно.`);
  }
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `HTTP ${response.status}`);
  }
  return response.json();
}

function renderFeed() {
  elements.feedCount.textContent = `${state.items.length}`;
  if (state.items.length === 0) {
    const title = state.feedEmptyTitle || "Лента пока пустая";
    const message = state.feedEmptyMessage || "Запусти поиск справа или проверь, что gateway-api и feed-service доступны.";
    elements.feed.innerHTML = `
      <section class="feed-empty">
        <h2>${escapeHTML(title)}</h2>
        <p>${escapeHTML(message)}</p>
      </section>
    `;
    return;
  }
  elements.feed.innerHTML = state.items.map(renderArticle).join("");
}

function renderFeedPaging(payload = {}) {
  const refillStarted = Boolean(payload.refill_started || payload.RefillStarted);
  const nextCursor = state.nextCursor;
  elements.loadMoreFeed.hidden = !state.feedQuery;
  elements.loadMoreFeed.disabled = state.feedLoadingMore || !nextCursor;
  elements.loadMoreFeed.textContent = state.feedLoadingMore
    ? "Загрузка"
    : nextCursor
      ? "Загрузить еще"
      : "Больше нет";
  const parts = [];
  parts.push(...buildFeedStatusParts({
    query: state.feedQuery,
    sources: state.feedSources,
    fromDate: state.feedFromDate,
    nextCursor,
    refillStarted,
    refillJobID: payload.refill_job?.ID ?? payload.RefillJob?.ID ?? payload.refill_job?.id ?? "",
  }));
  elements.feedPageStatus.textContent = parts.join(" | ");
}

function resetFeedPaging() {
  state.feedQuery = "";
  state.feedSources = [];
  state.feedLimit = 30;
  state.feedFromDate = "";
  state.feedToDate = "";
  state.nextCursor = "";
  state.feedLoadingMore = false;
  state.feedHasMore = false;
}

function appendUniqueFeedItems(items) {
  const seen = new Set(state.items.map((item) => item.id || item.url));
  for (const item of items) {
    const key = item.id || item.url;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    state.items.push(item);
  }
}

function renderArticle(item) {
  const tags = item.tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("");
  const reasons = item.scoreReasons.map((reason) => `<span>${escapeHTML(reason)}</span>`).join("");
  const date = formatDate(item.publishedAt);
  const reaction = item.reaction ? `<span class="reaction-chip">${escapeHTML(reactionLabel(item.reaction))}</span>` : "";
  return `
    <article class="feed-item" data-article-id="${escapeHTML(item.id)}">
      <div class="feed-item__meta">
        <span>${escapeHTML(item.sourceName || "source")}</span>
        <span>score ${formatScore(item.score)}</span>
        ${date ? `<span>${escapeHTML(date)}</span>` : ""}
        ${reaction}
      </div>
      <h2>${escapeHTML(item.title)}</h2>
      <p>${escapeHTML(stripHTML(item.summary)).slice(0, 420)}</p>
      <div class="tag-row">${tags}</div>
      ${reasons ? `<div class="reason-row">${reasons}</div>` : ""}
      <div class="article-actions">
        <button type="button" data-open-detail>Читать здесь</button>
        <a href="${escapeHTML(item.url)}" target="_blank" rel="noreferrer" data-reaction="open">Открыть</a>
        <button type="button" data-reaction="save">Сохранить</button>
        <button type="button" data-reaction="like">Нравится</button>
        <button type="button" data-reaction="skip">Пропустить</button>
        <button type="button" data-reaction="dislike">Не интересно</button>
      </div>
      <output class="reaction-state" aria-live="polite"></output>
    </article>
  `;
}

function renderArticleDetail() {
  const article = state.selectedArticle;
  if (!article && !state.articleLoadingID) {
    elements.articleDetail.hidden = true;
    elements.articleDetail.innerHTML = "";
    return;
  }
  elements.articleDetail.hidden = false;
  if (state.articleLoadingID && !article) {
    elements.articleDetail.innerHTML = `
      <div class="detail-head">
        <h2>Загрузка</h2>
        <button type="button" data-close-detail>Закрыть</button>
      </div>
      <p class="detail-muted">${escapeHTML(state.articleLoadingID)}</p>
    `;
    return;
  }

  const tags = article.tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("");
  const date = formatDate(article.publishedAt);
  const body = article.content || article.summary || "";
  elements.articleDetail.innerHTML = `
    <div class="detail-head">
      <h2>${escapeHTML(article.title)}</h2>
      <button type="button" data-close-detail>Закрыть</button>
    </div>
    <div class="feed-item__meta">
      <span>${escapeHTML(article.sourceName || "source")}</span>
      ${article.author ? `<span>${escapeHTML(article.author)}</span>` : ""}
      ${date ? `<span>${escapeHTML(date)}</span>` : ""}
    </div>
    <div class="tag-row">${tags}</div>
    <div class="detail-body">${sanitizeArticleHTML(body)}</div>
    ${article.url ? `<a class="detail-source" href="${escapeHTML(article.url)}" target="_blank" rel="noreferrer">Источник</a>` : ""}
  `;
}

function renderJob(job) {
  const status = job.Status ?? job.status ?? "unknown";
  const count = job.CandidatesCount ?? job.candidates_count ?? 0;
  const id = job.ID ?? job.id ?? "";
  elements.jobStatus.textContent = status;
  elements.jobStatus.dataset.status = String(status).toLowerCase();
  elements.jobMeta.textContent = id ? `job ${id} | candidates ${count}` : "нет активной задачи";
  renderSourceStats(job.SourceStats ?? job.source_stats ?? []);
}

function renderJobHistory(errorMessage = "") {
  if (errorMessage) {
    elements.jobHistory.innerHTML = `<p class="job-history__empty">${escapeHTML(errorMessage)}</p>`;
    return;
  }
  if (state.jobHistory.length === 0) {
    elements.jobHistory.innerHTML = `<p class="job-history__empty">история задач пустая</p>`;
    return;
  }
  elements.jobHistory.innerHTML = state.jobHistory
    .map((job) => {
      const status = String(job.Status ?? job.status ?? "unknown");
      const count = Number(job.CandidatesCount ?? job.candidates_count ?? 0);
      const query = job.Query?.Text ?? job.query?.text ?? "";
      const updatedAt = formatDate(job.UpdatedAt ?? job.updated_at ?? "");
      const sourceStats = job.SourceStats ?? job.source_stats ?? [];
      const sources = Array.isArray(sourceStats)
        ? sourceStats.map(normalizeSourceStats).map((stat) => stat.sourceName).filter(Boolean).join(", ")
        : "";
      return `
        <button class="job-history__item" type="button" data-job-id="${escapeHTML(job.ID ?? job.id ?? "")}">
          <span>${escapeHTML(status)}</span>
          <strong>${escapeHTML(query || "без запроса")}</strong>
          <small>${count} items${sources ? ` · ${escapeHTML(sources)}` : ""}${updatedAt ? ` · ${escapeHTML(updatedAt)}` : ""}</small>
        </button>
      `;
    })
    .join("");
}

function renderSourceStats(rawStats) {
  const stats = Array.isArray(rawStats) ? rawStats.map(normalizeSourceStats) : [];
  if (stats.length === 0) {
    elements.sourceStats.innerHTML = `<p class="source-stats__empty">нет статистики источников</p>`;
    return;
  }
  elements.sourceStats.innerHTML = stats
    .map((stat) => {
      const error = stat.error ? `<span class="source-stats__error">${escapeHTML(stat.error)}</span>` : "";
      const strategy = stat.strategy ? ` · ${escapeHTML(stat.strategy)}` : "";
      return `
        <div class="source-stat" data-status="${escapeHTML(stat.status || "unknown")}">
          <div>
            <strong>${escapeHTML(stat.sourceName || "source")}</strong>
            <span>${escapeHTML(stat.status || "unknown")}${strategy} · ${stat.durationMS} ms</span>
          </div>
          <dl>
            <dt>found</dt><dd>${stat.foundCount}</dd>
            <dt>accepted</dt><dd>${stat.acceptedCount}</dd>
            <dt>returned</dt><dd>${stat.returnedCount}</dd>
            <dt>filtered</dt><dd>${stat.filteredCount}</dd>
          </dl>
          ${error}
        </div>
      `;
    })
    .join("");
}

function selectedSources() {
  return buildSelectedSources({
    allSelected: elements.sourceAll.checked,
    selected: sourceOptions().filter((option) => option.checked).map((option) => option.value),
  });
}

function selectedFreshnessRange() {
  return buildFreshnessRange(elements.searchFreshness?.value || "all");
}

function syncSourceControls() {
  if (elements.sourceAll.checked) {
    for (const option of sourceOptions()) {
      option.checked = false;
    }
    return;
  }
  const hasSelectedSource = sourceOptions().some((option) => option.checked);
  if (!hasSelectedSource) {
    elements.sourceAll.checked = true;
  }
}

function renderSourceControls() {
  const sources = state.parserSources.length > 0 ? state.parserSources : fallbackSources();
  const selected = new Set(selectedSources());
  elements.sourceControls.querySelectorAll("[data-source-option-label]").forEach((node) => node.remove());
  const fragment = document.createDocumentFragment();
  for (const source of sources) {
    const label = document.createElement("label");
    label.className = "source-option";
    label.dataset.sourceOptionLabel = "true";
    const input = document.createElement("input");
    input.type = "checkbox";
    input.value = source.name;
    input.dataset.sourceOption = "true";
    input.disabled = !source.enabled || !source.searchable;
    input.checked = !elements.sourceAll.checked && selected.has(source.name);
    const text = document.createElement("span");
    text.textContent = source.displayName || source.name;
    if (!source.enabled) {
      text.textContent += " off";
    }
    label.append(input, text);
    fragment.append(label);
  }
  elements.sourceControls.append(fragment);
}

function sourceOptions() {
  return [...elements.sourceControls.querySelectorAll("[data-source-option]")];
}

function fallbackSources() {
  return [
    { name: "habr", displayName: "Habr", enabled: true, searchable: true },
    { name: "vc", displayName: "vc.ru", enabled: true, searchable: true },
    { name: "dzen", displayName: "Яндекс Дзен", enabled: true, searchable: true },
    { name: "vc_rss", displayName: "vc RSS", enabled: true, searchable: true },
  ];
}

function markReaction(articleID, type) {
  const output = elements.feed.querySelector(`[data-article-id="${cssEscape(articleID)}"] .reaction-state`);
  if (output) {
    output.textContent = reactionLabel(type);
  }
}

function reactionLabel(type) {
  switch (type) {
    case "save":
      return "Сохранено";
    case "like":
      return "Нравится";
    case "dislike":
      return "Не интересно";
    case "open":
      return "Открыто";
    case "skip":
      return "Скрыто";
    default:
      return `Отправлено: ${type}`;
  }
}

function setError(message) {
  elements.lastError.textContent = message || "";
  elements.lastError.hidden = !message;
}

function trimTrailingSlash(value) {
  return String(value || "").trim().replace(/\/+$/, "");
}

function stripHTML(value) {
  return String(value || "").replace(/<[^>]*>/g, " ").replace(/\s+/g, " ").trim();
}

function sanitizeArticleHTML(value) {
  const template = document.createElement("template");
  template.innerHTML = String(value || "");
  for (const element of template.content.querySelectorAll("script,style,iframe,object,embed")) {
    element.remove();
  }
  for (const element of template.content.querySelectorAll("*")) {
    for (const attribute of [...element.attributes]) {
      const name = attribute.name.toLowerCase();
      const val = attribute.value.trim().toLowerCase();
      if (name.startsWith("on") || val.startsWith("javascript:")) {
        element.removeAttribute(attribute.name);
      }
    }
  }
  return template.innerHTML || `<p>${escapeHTML(stripHTML(value))}</p>`;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function cssEscape(value) {
  if (window.CSS?.escape) {
    return window.CSS.escape(value);
  }
  return String(value).replaceAll('"', '\\"');
}

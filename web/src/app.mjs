import {
  buildReactionPayload,
  buildSearchJobPayload,
  formatDate,
  formatScore,
  normalizeArticle,
  normalizeCandidateItem,
  normalizeJobResponse,
  normalizeFeedItem,
  shouldPollJob,
} from "./app-core.mjs";

const state = {
  apiBase: localStorage.getItem("articleflow.apiBase") || "http://localhost:8080",
  userID: localStorage.getItem("articleflow.userID") || "reader-demo",
  items: [],
  selectedIndex: 0,
  activeJob: null,
  selectedArticle: null,
  articleLoadingID: "",
  pollTimer: 0,
  feedEmptyTitle: "",
  feedEmptyMessage: "",
};

const elements = {
  apiBase: document.querySelector("#apiBase"),
  userID: document.querySelector("#userID"),
  feed: document.querySelector("#feed"),
  feedCount: document.querySelector("#feedCount"),
  refreshFeed: document.querySelector("#refreshFeed"),
  searchForm: document.querySelector("#searchForm"),
  searchQuery: document.querySelector("#searchQuery"),
  searchLimit: document.querySelector("#searchLimit"),
  jobStatus: document.querySelector("#jobStatus"),
  jobMeta: document.querySelector("#jobMeta"),
  lastError: document.querySelector("#lastError"),
  articleDetail: document.querySelector("#articleDetail"),
};

elements.apiBase.value = state.apiBase;
elements.userID.value = state.userID;

elements.apiBase.addEventListener("change", () => {
  state.apiBase = trimTrailingSlash(elements.apiBase.value);
  elements.apiBase.value = state.apiBase;
  localStorage.setItem("articleflow.apiBase", state.apiBase);
});

elements.userID.addEventListener("change", () => {
  state.userID = elements.userID.value.trim() || "reader-demo";
  elements.userID.value = state.userID;
  localStorage.setItem("articleflow.userID", state.userID);
});

elements.refreshFeed.addEventListener("click", () => {
  void loadFeed();
});

elements.searchForm.addEventListener("submit", (event) => {
  event.preventDefault();
  void startSearchJob();
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

elements.articleDetail.addEventListener("click", (event) => {
  if (event.target.closest("[data-close-detail]")) {
    state.selectedArticle = null;
    renderArticleDetail();
  }
});

void loadFeed();

async function loadFeed() {
  setError("");
  elements.refreshFeed.disabled = true;
  try {
    const payload = await requestJSON(`/api/v1/feed?limit=30`);
    state.items = (payload.items || payload.Items || []).map(normalizeFeedItem);
    state.feedEmptyTitle = "Лента пока пустая";
    state.feedEmptyMessage = "Запусти поиск справа или проверь, что gateway-api и feed-service доступны.";
    renderFeed();
  } catch (error) {
    setError(error.message);
  } finally {
    elements.refreshFeed.disabled = false;
  }
}

async function startSearchJob() {
  setError("");
  clearTimeout(state.pollTimer);
  try {
    const searchPayload = buildSearchJobPayload({
      query: elements.searchQuery.value,
      sources: ["habr", "vc"],
      limit: elements.searchLimit.value,
    });
    const jobPayload = await requestJSON("/api/v1/search/jobs", {
      method: "POST",
      body: JSON.stringify(searchPayload),
    });
    state.activeJob = normalizeJobResponse(jobPayload);
    renderJob(state.activeJob);
    scheduleJobPoll();
  } catch (error) {
    setError(error.message);
  }
}

function scheduleJobPoll() {
  clearTimeout(state.pollTimer);
  if (!shouldPollJob(state.activeJob)) {
    if (String(state.activeJob?.Status ?? state.activeJob?.status ?? "") === "completed") {
      renderCompletedJobResults(state.activeJob);
    }
    return;
  }
  state.pollTimer = window.setTimeout(async () => {
    try {
      const jobID = state.activeJob.ID ?? state.activeJob.id;
      const payload = await requestJSON(`/api/v1/search/jobs/${encodeURIComponent(jobID)}`);
      state.activeJob = normalizeJobResponse(payload);
      renderJob(state.activeJob);
      scheduleJobPoll();
    } catch (error) {
      setError(error.message);
    }
  }, 1200);
}

function renderCompletedJobResults(job) {
  const candidates = job.Candidates ?? job.candidates ?? [];
  state.items = candidates.map(normalizeCandidateItem);
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
}

async function sendReaction(articleID, type) {
  setError("");
  try {
    const payload = buildReactionPayload({ userID: state.userID, articleID, type });
    await requestJSON("/api/v1/reactions", {
      method: "POST",
      body: JSON.stringify(payload),
    });
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

function renderArticle(item) {
  const tags = item.tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join("");
  const date = formatDate(item.publishedAt);
  return `
    <article class="feed-item" data-article-id="${escapeHTML(item.id)}">
      <div class="feed-item__meta">
        <span>${escapeHTML(item.sourceName || "source")}</span>
        <span>score ${formatScore(item.score)}</span>
        ${date ? `<span>${escapeHTML(date)}</span>` : ""}
      </div>
      <h2>${escapeHTML(item.title)}</h2>
      <p>${escapeHTML(stripHTML(item.summary)).slice(0, 420)}</p>
      <div class="tag-row">${tags}</div>
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
}

function markReaction(articleID, type) {
  const output = elements.feed.querySelector(`[data-article-id="${cssEscape(articleID)}"] .reaction-state`);
  if (output) {
    output.textContent = `Отправлено: ${type}`;
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

import {
  buildReactionPayload,
  buildSearchJobPayload,
  formatDate,
  formatScore,
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
  pollTimer: 0,
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
  const button = event.target.closest("[data-reaction]");
  if (!button) {
    return;
  }
  const articleID = button.closest("[data-article-id]")?.dataset.articleId;
  void sendReaction(articleID, button.dataset.reaction);
});

void loadFeed();

async function loadFeed() {
  setError("");
  elements.refreshFeed.disabled = true;
  try {
    const payload = await requestJSON(`/api/v1/feed?limit=30`);
    state.items = (payload.items || payload.Items || []).map(normalizeFeedItem);
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
      sources: ["habr"],
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
      void loadFeed();
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
    elements.feed.innerHTML = `
      <section class="feed-empty">
        <h2>Лента пока пустая</h2>
        <p>Запусти поиск Habr справа или проверь, что gateway-api и feed-service доступны.</p>
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

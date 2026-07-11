package dzen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

const SourceName = "dzen"
const maxSearchPages = 4
const maxDzenArticleAge = 5 * 365 * 24 * time.Hour

type ClientOptions struct {
	BaseURL       string
	PublicBaseURL string
	HTTPClient    *http.Client
	Language      string
}

type Client struct {
	baseURL       string
	publicBaseURL string
	client        *http.Client
	language      string
	now           func() time.Time
}

func NewClient(options ClientOptions) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://dzen.ru"
	}
	publicBaseURL := strings.TrimRight(strings.TrimSpace(options.PublicBaseURL), "/")
	if publicBaseURL == "" {
		publicBaseURL = baseURL
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:       baseURL,
		publicBaseURL: publicBaseURL,
		client:        httpClient,
		language:      firstNonEmpty(options.Language, "ru"),
		now:           time.Now,
	}
}

func (client *Client) SourceName() string {
	return SourceName
}

func (client *Client) Strategy() string {
	return "html_search"
}

func (client *Client) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	query = query.Normalize()
	searchBody, finalURL, err := client.get(ctx, client.searchURL(query.Text))
	if err != nil {
		return nil, err
	}
	defer searchBody.Close()
	if isAuthRedirect(finalURL) {
		return nil, fmt.Errorf("dzen search redirected to auth: %s", finalURL)
	}

	urls, err := client.collectSearchURLs(ctx, searchBody, finalURL, query.Limit)
	if err != nil {
		return nil, err
	}
	return client.fetchCandidates(ctx, urls, query.Limit), nil
}

func (client *Client) collectSearchURLs(ctx context.Context, searchBody io.Reader, finalURL string, limit int) ([]string, error) {
	targetURLCount := searchURLTarget(limit)
	page, err := ParseSearchPage(searchBody, finalURL, client.publicBaseURL, targetURLCount)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, targetURLCount)
	seen := make(map[string]bool, targetURLCount)
	addURLs := func(nextURLs []string) {
		for _, articleURL := range nextURLs {
			if articleURL == "" || seen[articleURL] {
				continue
			}
			seen[articleURL] = true
			urls = append(urls, articleURL)
			if targetURLCount > 0 && len(urls) >= targetURLCount {
				return
			}
		}
	}
	addURLs(page.URLs)
	moreURL := page.MoreURL
	for pageIndex := 1; moreURL != "" && (targetURLCount <= 0 || len(urls) < targetURLCount) && pageIndex < maxSearchPages; pageIndex++ {
		moreBody, moreFinalURL, err := client.get(ctx, client.fetchURL(moreURL))
		if err != nil {
			break
		}
		nextPage, parseErr := ParseSearchPage(moreBody, moreFinalURL, client.publicBaseURL, targetURLCount-len(urls))
		closeErr := moreBody.Close()
		if parseErr != nil || closeErr != nil {
			break
		}
		addURLs(nextPage.URLs)
		moreURL = nextPage.MoreURL
	}
	return urls, nil
}

func (client *Client) fetchCandidates(ctx context.Context, urls []string, limit int) []parserv1.ArticleCandidate {
	if len(urls) == 0 {
		return nil
	}
	workers := minInt(4, len(urls))
	type job struct {
		index int
		url   string
	}
	type result struct {
		index     int
		candidate parserv1.ArticleCandidate
		ok        bool
	}
	jobs := make(chan job)
	results := make(chan result, len(urls))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				candidate, ok := client.fetchCandidate(ctx, item.url)
				results <- result{index: item.index, candidate: candidate, ok: ok}
			}
		}()
	}
	for index, articleURL := range urls {
		jobs <- job{index: index, url: articleURL}
	}
	close(jobs)
	wg.Wait()
	close(results)

	ordered := make([]result, len(urls))
	for item := range results {
		ordered[item.index] = item
	}
	candidates := make([]parserv1.ArticleCandidate, 0, len(urls))
	for _, item := range ordered {
		if !item.ok {
			continue
		}
		if client.isStaleArticle(item.candidate.PublishedAt) {
			continue
		}
		candidates = append(candidates, item.candidate)
		if limit > 0 && len(candidates) >= limit {
			break
		}
	}
	return candidates
}

func (client *Client) fetchCandidate(ctx context.Context, articleURL string) (parserv1.ArticleCandidate, bool) {
	articleBody, _, err := client.get(ctx, client.fetchURL(articleURL))
	if err != nil {
		return parserv1.ArticleCandidate{}, false
	}
	article, parseErr := ParseArticleHTML(articleBody, articleURL)
	closeErr := articleBody.Close()
	if parseErr != nil || closeErr != nil {
		return parserv1.ArticleCandidate{}, false
	}
	return parserv1.ArticleCandidate{
		SourceName:  SourceName,
		ExternalID:  article.ExternalID,
		URL:         article.URL,
		Title:       article.Title,
		Summary:     article.Summary,
		Content:     article.Content,
		Author:      article.Author,
		Tags:        article.Tags,
		Language:    firstNonEmpty(article.Language, client.language),
		PublishedAt: article.PublishedAt,
	}, true
}

func (client *Client) isStaleArticle(publishedAt time.Time) bool {
	if publishedAt.IsZero() {
		return false
	}
	now := time.Now
	if client.now != nil {
		now = client.now
	}
	age := now().UTC().Sub(publishedAt.UTC())
	return age > maxDzenArticleAge
}

func (client *Client) fetchURL(articleURL string) string {
	base, baseErr := url.Parse(client.baseURL)
	publicBase, publicErr := url.Parse(client.publicBaseURL)
	parsed, parsedErr := url.Parse(articleURL)
	if baseErr != nil || publicErr != nil || parsedErr != nil || parsed.Host != publicBase.Host {
		return articleURL
	}
	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	return parsed.String()
}

func (client *Client) searchURL(query string) string {
	parsed, _ := url.Parse(client.baseURL + "/search")
	values := parsed.Query()
	values.Set("query", strings.TrimSpace(query))
	values.Set("type_filter", "article,brief")
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func (client *Client) get(ctx context.Context, requestURL string) (io.ReadCloser, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
	request.Header.Set("Cookie", "zen_sso_checked=1; zen_vk_sso_checked=1")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126 Safari/537.36 articleflow-parser/0.1")
	response, err := client.client.Do(request)
	if err != nil {
		return nil, "", err
	}
	finalURL := requestURL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		if response.Header.Get("Location") != "" {
			return nil, finalURL, fmt.Errorf("GET %s returned status %d redirecting to %s", requestURL, response.StatusCode, response.Header.Get("Location"))
		}
		return nil, finalURL, fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
	}
	return response.Body, finalURL, nil
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func searchURLTarget(limit int) int {
	if limit <= 0 {
		return 50
	}
	return maxInt(limit*2, limit+10)
}

func isAuthRedirect(finalURL string) bool {
	parsed, err := url.Parse(finalURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return strings.Contains(host, "passport.yandex.") || strings.Contains(host, "login.vk.com")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

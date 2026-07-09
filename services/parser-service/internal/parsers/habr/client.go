package habr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Waiter interface {
	Wait(ctx context.Context) error
}

type NoopWaiter struct{}

func (NoopWaiter) Wait(context.Context) error {
	return nil
}

type FixedDelayWaiter struct {
	Delay time.Duration
}

func (waiter FixedDelayWaiter) Wait(ctx context.Context) error {
	if waiter.Delay <= 0 {
		return nil
	}
	timer := time.NewTimer(waiter.Delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type ClientOptions struct {
	BaseURL     string
	HTTPClient  *http.Client
	MaxAttempts int
	RetryDelay  time.Duration
	Waiter      Waiter
}

type Client struct {
	baseURL     string
	httpClient  *http.Client
	maxAttempts int
	retryDelay  time.Duration
	waiter      Waiter
}

func NewClient(options ClientOptions) *Client {
	base := strings.TrimRight(options.BaseURL, "/")
	if base == "" {
		base = baseURL
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	maxAttempts := options.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	waiter := options.Waiter
	if waiter == nil {
		waiter = FixedDelayWaiter{Delay: 500 * time.Millisecond}
	}
	return &Client{
		baseURL:     base,
		httpClient:  httpClient,
		maxAttempts: maxAttempts,
		retryDelay:  options.RetryDelay,
		waiter:      waiter,
	}
}

func (client *Client) SourceName() string {
	return SourceName
}

func (client *Client) Strategy() string {
	return "rss_search_html_article"
}

func (client *Client) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return nil, err
	}
	searchBody, err := client.get(ctx, client.searchURL(query))
	if err != nil {
		return nil, err
	}
	defer searchBody.Close()

	events, err := ParseRSS(searchBody)
	if err != nil {
		return nil, err
	}
	candidates := candidatesFromEvents(events, query)
	for index := range candidates {
		articleBody, err := client.get(ctx, client.sourceURL(candidates[index].URL))
		if err != nil {
			return nil, err
		}
		article, parseErr := ParseArticleHTML(articleBody, candidates[index].URL)
		closeErr := articleBody.Close()
		if parseErr != nil {
			return nil, parseErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		candidates[index].Title = firstNonEmpty(article.Title, candidates[index].Title)
		candidates[index].Summary = firstNonEmpty(candidates[index].Summary, article.Content)
		candidates[index].Content = article.Content
		candidates[index].Author = firstNonEmpty(article.Author, candidates[index].Author)
		if len(article.Tags) > 0 {
			candidates[index].Tags = article.Tags
		}
		if !article.PublishedAt.IsZero() {
			candidates[index].PublishedAt = article.PublishedAt
		}
	}
	return candidates, nil
}

func (client *Client) searchURL(query parserv1.SearchQuery) string {
	values := url.Values{}
	values.Set("q", query.Text)
	values.Set("target_type", "posts")
	values.Set("order_by", "relevance")
	values.Set("hl", query.Language)
	values.Set("fl", query.Language)
	return client.baseURL + "/ru/rss/search/?" + values.Encode()
}

func (client *Client) sourceURL(articleURL string) string {
	parsed, err := url.Parse(articleURL)
	if err != nil || parsed.Path == "" {
		return articleURL
	}
	return client.baseURL + parsed.RequestURI()
}

func candidatesFromEvents(events []eventsv1.ArticleDiscoveredEvent, query parserv1.SearchQuery) []parserv1.ArticleCandidate {
	candidates := make([]parserv1.ArticleCandidate, 0, len(events))
	for _, event := range events {
		candidate := parserv1.ArticleCandidate{
			SourceName:  event.SourceName,
			ExternalID:  event.ExternalID,
			URL:         event.URL,
			Title:       event.Title,
			Summary:     event.Summary,
			Author:      event.Author,
			Tags:        event.Tags,
			Language:    event.Language,
			PublishedAt: event.PublishedAt,
		}
		if candidate.Language == "" {
			candidate.Language = query.Language
		}
		candidates = append(candidates, candidate)
	}
	if query.Limit > 0 && len(candidates) > query.Limit {
		candidates = candidates[:query.Limit]
	}
	return candidates
}

func (client *Client) get(ctx context.Context, requestURL string) (io.ReadCloser, error) {
	var lastErr error
	for attempt := 1; attempt <= client.maxAttempts; attempt++ {
		if err := client.waiter.Wait(ctx); err != nil {
			return nil, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("User-Agent", "articleflow-parser/0.1")

		response, err := client.httpClient.Do(request)
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			return response.Body, nil
		}
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
		}
		if attempt < client.maxAttempts && client.retryDelay > 0 {
			timer := time.NewTimer(client.retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, lastErr
}

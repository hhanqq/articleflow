package vc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/rssfeed"
)

const SourceName = "vc"

type ClientOptions struct {
	BaseURL    string
	HTTPClient *http.Client
	Language   string
}

type Client struct {
	fallback *rssfeed.Client
	baseURL  string
	client   *http.Client
}

func NewClient(options ClientOptions) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://vc.ru"
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL: baseURL,
		client:  httpClient,
		fallback: rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: SourceName,
			FeedURL:    baseURL + "/rss",
			HTTPClient: httpClient,
			Language:   firstNonEmpty(options.Language, "ru"),
		}),
	}
}

func (client *Client) SourceName() string {
	return SourceName
}

func (client *Client) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	candidates, err := client.fallback.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	for index := range candidates {
		articleBody, err := client.get(ctx, candidates[index].URL)
		if err != nil {
			continue
		}
		article, parseErr := ParseArticleHTML(articleBody, candidates[index].URL)
		closeErr := articleBody.Close()
		if parseErr != nil || closeErr != nil {
			continue
		}
		candidates[index].URL = firstNonEmpty(article.URL, candidates[index].URL)
		candidates[index].Title = firstNonEmpty(article.Title, candidates[index].Title)
		candidates[index].Summary = firstNonEmpty(article.Summary, candidates[index].Summary)
		candidates[index].Content = firstNonEmpty(article.Content, candidates[index].Content)
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

func (client *Client) get(ctx context.Context, requestURL string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "articleflow-parser/0.1")
	response, err := client.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return nil, fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
	}
	return response.Body, nil
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

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
	BaseURL     string
	HTTPClient  *http.Client
	Language    string
	URLSearcher URLSearcher
}

type URLSearcher interface {
	SearchURLs(ctx context.Context, query string, limit int) ([]string, error)
}

type Client struct {
	fallback    *rssfeed.Client
	baseURL     string
	client      *http.Client
	language    string
	urlSearcher URLSearcher
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
		baseURL:     baseURL,
		client:      httpClient,
		language:    firstNonEmpty(options.Language, "ru"),
		urlSearcher: options.URLSearcher,
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

func (client *Client) Strategy() string {
	if client.urlSearcher == nil {
		return "rss_fallback"
	}
	return "html_search_with_rss_fallback"
}

func (client *Client) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if client.urlSearcher != nil {
		candidates, err := client.searchURLs(ctx, query.Normalize())
		if err == nil && len(candidates) > 0 {
			return candidates, nil
		}
		fallbackCandidates, fallbackErr := client.searchRSS(ctx, query)
		if fallbackErr == nil && (len(fallbackCandidates) > 0 || err == nil) {
			return fallbackCandidates, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, fallbackErr
	}
	return client.searchRSS(ctx, query)
}

func (client *Client) searchRSS(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
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

func (client *Client) searchURLs(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	urls, err := client.urlSearcher.SearchURLs(ctx, query.Text, query.Limit)
	if err != nil {
		return nil, err
	}
	candidates := make([]parserv1.ArticleCandidate, 0, len(urls))
	for _, articleURL := range urls {
		articleBody, err := client.get(ctx, articleURL)
		if err != nil {
			continue
		}
		article, parseErr := ParseArticleHTML(articleBody, articleURL)
		closeErr := articleBody.Close()
		if parseErr != nil || closeErr != nil {
			continue
		}
		candidates = append(candidates, parserv1.ArticleCandidate{
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
		})
		if query.Limit > 0 && len(candidates) >= query.Limit {
			break
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

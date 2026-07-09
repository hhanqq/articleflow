package dzen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

const SourceName = "dzen"

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
	}
}

func (client *Client) SourceName() string {
	return SourceName
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

	urls, err := ParseSearchHTML(searchBody, finalURL, client.publicBaseURL, query.Limit)
	if err != nil {
		return nil, err
	}
	candidates := make([]parserv1.ArticleCandidate, 0, len(urls))
	for _, articleURL := range urls {
		articleBody, _, err := client.get(ctx, client.fetchURL(articleURL))
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
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func (client *Client) get(ctx context.Context, requestURL string) (io.ReadCloser, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("User-Agent", "Mozilla/5.0 articleflow-parser/0.1")
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

package vc

import (
	"context"
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
}

func NewClient(options ClientOptions) *Client {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://vc.ru"
	}
	return &Client{
		fallback: rssfeed.NewClient(rssfeed.ClientOptions{
			SourceName: SourceName,
			FeedURL:    baseURL + "/rss",
			HTTPClient: options.HTTPClient,
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
	return client.fallback.Search(ctx, query)
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

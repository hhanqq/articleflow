package articlev1

import (
	"errors"
	"strings"
	"time"
)

type Article struct {
	ID          string
	SourceID    string
	SourceName  string
	ExternalID  string
	URL         string
	Title       string
	Summary     string
	Content     string
	Author      string
	Tags        []string
	Language    string
	PublishedAt time.Time
	ParsedAt    time.Time
}

func (article Article) Validate() error {
	if strings.TrimSpace(article.URL) == "" {
		return errors.New("url is required")
	}
	if strings.TrimSpace(article.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(article.SourceName) == "" {
		return errors.New("source name is required")
	}
	return nil
}

type ArticlePreview struct {
	ID          string
	SourceName  string
	URL         string
	Title       string
	Summary     string
	Tags        []string
	PublishedAt time.Time
}

type SearchQuery struct {
	Text    string
	Sources []string
	Limit   int
}

func (query SearchQuery) Normalize() SearchQuery {
	query.Text = strings.TrimSpace(query.Text)
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	return query
}

func (query SearchQuery) Validate() error {
	query = query.Normalize()
	if query.Text == "" {
		return errors.New("search query text is required")
	}
	return nil
}

type Source struct {
	ID        string
	Name      string
	BaseURL   string
	Enabled   bool
	CreatedAt time.Time
}

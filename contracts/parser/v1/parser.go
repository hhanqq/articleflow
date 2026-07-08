package parserv1

import (
	"errors"
	"strings"
	"time"
)

const (
	ParserJobStatusQueued    = "queued"
	ParserJobStatusRunning   = "running"
	ParserJobStatusCompleted = "completed"
	ParserJobStatusFailed    = "failed"
)

type SearchQuery struct {
	Text     string
	Sources  []string
	Tags     []string
	Limit    int
	Offset   int
	Language string
	FromDate *time.Time
	ToDate   *time.Time
}

func (query SearchQuery) Normalize() SearchQuery {
	query.Text = strings.TrimSpace(query.Text)
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if strings.TrimSpace(query.Language) == "" {
		query.Language = "ru"
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

type ArticleCandidate struct {
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
}

type ParserJob struct {
	ID              string
	Query           SearchQuery
	Status          string
	Error           string
	CandidatesCount int
	Candidates      []ArticleCandidate
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

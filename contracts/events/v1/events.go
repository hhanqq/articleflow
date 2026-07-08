package eventsv1

import (
	"errors"
	"strings"
	"time"
)

const (
	TopicArticleDiscovered   = "article.discovered.v1"
	TopicArticleCreated      = "article.created.v1"
	TopicArticleUpdated      = "article.updated.v1"
	TopicUserReactionCreated = "user.reaction.created.v1"
	TopicFeedItemScored      = "feed.item.scored.v1"
	TopicParserJobFailed     = "parser.job.failed.v1"
)

type ArticleDiscoveredEvent struct {
	SourceName   string
	ExternalID   string
	URL          string
	Title        string
	Summary      string
	Content      string
	Author       string
	Tags         []string
	Language     string
	PublishedAt  time.Time
	DiscoveredAt time.Time
}

func (event ArticleDiscoveredEvent) Validate() error {
	if strings.TrimSpace(event.SourceName) == "" {
		return errors.New("source name is required")
	}
	if strings.TrimSpace(event.ExternalID) == "" {
		return errors.New("external id is required")
	}
	if strings.TrimSpace(event.URL) == "" {
		return errors.New("url is required")
	}
	if strings.TrimSpace(event.Title) == "" {
		return errors.New("title is required")
	}
	if event.DiscoveredAt.IsZero() {
		return errors.New("discovered at is required")
	}
	return nil
}

type ArticleCreatedEvent struct {
	ArticleID   string
	SourceName  string
	ExternalID  string
	URL         string
	Title       string
	CreatedAt   time.Time
	PublishedAt time.Time
}

type UserReactionCreatedEvent struct {
	UserID    string
	ArticleID string
	Type      string
	CreatedAt time.Time
}

func (event UserReactionCreatedEvent) Validate() error {
	if strings.TrimSpace(event.UserID) == "" {
		return errors.New("user id is required")
	}
	if strings.TrimSpace(event.ArticleID) == "" {
		return errors.New("article id is required")
	}
	if strings.TrimSpace(event.Type) == "" {
		return errors.New("reaction type is required")
	}
	if event.CreatedAt.IsZero() {
		return errors.New("created at is required")
	}
	return nil
}

type ParserJobFailedEvent struct {
	JobID      string
	SourceName string
	Query      string
	Error      string
	FailedAt   time.Time
}

type FeedItemScoredEvent struct {
	ArticleID    string
	SourceName   string
	URL          string
	Title        string
	Summary      string
	Tags         []string
	Score        float64
	ScoreReasons []string
	PublishedAt  time.Time
	ScoredAt     time.Time
}

func (event FeedItemScoredEvent) Validate() error {
	if strings.TrimSpace(event.ArticleID) == "" {
		return errors.New("article id is required")
	}
	if strings.TrimSpace(event.SourceName) == "" {
		return errors.New("source name is required")
	}
	if strings.TrimSpace(event.URL) == "" {
		return errors.New("url is required")
	}
	if strings.TrimSpace(event.Title) == "" {
		return errors.New("title is required")
	}
	if event.ScoredAt.IsZero() {
		return errors.New("scored at is required")
	}
	return nil
}

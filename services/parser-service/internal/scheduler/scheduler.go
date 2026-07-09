package scheduler

import (
	"context"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type JobStarter interface {
	StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error)
}

type Query struct {
	Text    string
	Sources []string
	Limit   int
}

type Config struct {
	Enabled  bool
	Queries  []Query
	Interval time.Duration
}

type Scheduler struct {
	starter JobStarter
	config  Config
}

func New(starter JobStarter, config Config) *Scheduler {
	return &Scheduler{starter: starter, config: config}
}

func (scheduler *Scheduler) Run(ctx context.Context) error {
	if !scheduler.config.Enabled || scheduler.starter == nil {
		return nil
	}
	if err := scheduler.RunOnce(ctx); err != nil {
		return err
	}
	interval := scheduler.config.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := scheduler.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (scheduler *Scheduler) RunOnce(ctx context.Context) error {
	if !scheduler.config.Enabled || scheduler.starter == nil {
		return nil
	}
	for _, configuredQuery := range scheduler.config.Queries {
		query := parserv1.SearchQuery{
			Text:    strings.TrimSpace(configuredQuery.Text),
			Sources: cleanSources(configuredQuery.Sources),
			Limit:   configuredQuery.Limit,
		}.Normalize()
		if err := query.Validate(); err != nil {
			continue
		}
		if _, err := scheduler.starter.StartAsync(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func cleanSources(values []string) []string {
	sources := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		source := strings.TrimSpace(value)
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	return sources
}

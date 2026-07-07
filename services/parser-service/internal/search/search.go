package search

import (
	"context"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type Parser interface {
	SourceName() string
	Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error)
}

type Usecase struct {
	producer articleflowkafka.Producer
	parsers  map[string]Parser
}

func NewUsecase(producer articleflowkafka.Producer, parsers []Parser) *Usecase {
	bySource := make(map[string]Parser, len(parsers))
	for _, parser := range parsers {
		bySource[parser.SourceName()] = parser
	}
	return &Usecase{producer: producer, parsers: bySource}
}

func (usecase *Usecase) SearchAndPublish(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return nil, err
	}

	var allCandidates []parserv1.ArticleCandidate
	for _, source := range selectedSources(query, usecase.parsers) {
		parser := usecase.parsers[source]
		candidates, err := parser.Search(ctx, query)
		if err != nil {
			if publishErr := usecase.publishFailure(ctx, source, query, err); publishErr != nil {
				return nil, publishErr
			}
			return nil, err
		}
		for _, candidate := range candidates {
			if err := usecase.publishCandidate(ctx, candidate); err != nil {
				return nil, err
			}
			allCandidates = append(allCandidates, candidate)
		}
	}
	if query.Limit > 0 && len(allCandidates) > query.Limit {
		allCandidates = allCandidates[:query.Limit]
	}
	return allCandidates, nil
}

func (usecase *Usecase) publishCandidate(ctx context.Context, candidate parserv1.ArticleCandidate) error {
	event := eventsv1.ArticleDiscoveredEvent{
		SourceName:   candidate.SourceName,
		ExternalID:   candidate.ExternalID,
		URL:          candidate.URL,
		Title:        candidate.Title,
		Summary:      candidate.Summary,
		Content:      candidate.Content,
		Author:       candidate.Author,
		Tags:         candidate.Tags,
		Language:     candidate.Language,
		PublishedAt:  candidate.PublishedAt,
		DiscoveredAt: nowUTC(),
	}
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		return err
	}
	return usecase.producer.Publish(ctx, articleflowkafka.Message{
		Topic: eventsv1.TopicArticleDiscovered,
		Key:   event.ExternalID,
		Value: payload,
	})
}

func (usecase *Usecase) publishFailure(ctx context.Context, source string, query parserv1.SearchQuery, cause error) error {
	event := eventsv1.ParserJobFailedEvent{
		JobID:      source + ":" + query.Text,
		SourceName: source,
		Query:      query.Text,
		Error:      cause.Error(),
		FailedAt:   nowUTC(),
	}
	payload, err := articleflowkafka.MarshalJSON(event)
	if err != nil {
		return err
	}
	return usecase.producer.Publish(ctx, articleflowkafka.Message{
		Topic: eventsv1.TopicParserJobFailed,
		Key:   event.JobID,
		Value: payload,
	})
}

func selectedSources(query parserv1.SearchQuery, parsers map[string]Parser) []string {
	if len(query.Sources) > 0 {
		return query.Sources
	}
	sources := make([]string, 0, len(parsers))
	for source := range parsers {
		sources = append(sources, source)
	}
	return sources
}

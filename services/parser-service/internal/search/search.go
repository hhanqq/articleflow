package search

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strings"

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
	seenCandidates := make(map[string]bool)
	var lastErr error
	for _, source := range selectedSources(query, usecase.parsers) {
		parser := usecase.parsers[source]
		candidates, err := parser.Search(ctx, query)
		if err != nil {
			if publishErr := usecase.publishFailure(ctx, source, query, err); publishErr != nil {
				return nil, publishErr
			}
			lastErr = err
			continue
		}
		for _, candidate := range candidates {
			candidate = normalizeCandidate(candidate, source, query)
			if candidate.URL == "" || candidate.Title == "" {
				continue
			}
			key := candidateDedupKey(candidate)
			if seenCandidates[key] {
				continue
			}
			seenCandidates[key] = true
			if err := usecase.publishCandidate(ctx, candidate); err != nil {
				return nil, err
			}
			allCandidates = append(allCandidates, candidate)
		}
	}
	if len(allCandidates) == 0 && lastErr != nil {
		return nil, lastErr
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

func normalizeCandidate(candidate parserv1.ArticleCandidate, source string, query parserv1.SearchQuery) parserv1.ArticleCandidate {
	candidate.SourceName = strings.TrimSpace(candidate.SourceName)
	if candidate.SourceName == "" {
		candidate.SourceName = strings.TrimSpace(source)
	}
	candidate.ExternalID = strings.TrimSpace(candidate.ExternalID)
	candidate.URL = normalizeCandidateURL(candidate.URL)
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.Summary = strings.TrimSpace(candidate.Summary)
	candidate.Content = strings.TrimSpace(candidate.Content)
	candidate.Author = strings.TrimSpace(candidate.Author)
	candidate.Language = strings.TrimSpace(candidate.Language)
	if candidate.Language == "" {
		candidate.Language = query.Language
	}
	candidate.Tags = cleanCandidateTags(candidate.Tags)
	if candidate.ExternalID == "" {
		candidate.ExternalID = stableCandidateID(candidate.URL)
	}
	return candidate
}

func candidateDedupKey(candidate parserv1.ArticleCandidate) string {
	if candidate.URL != "" {
		return "url:" + strings.ToLower(candidate.URL)
	}
	return "source:" + strings.ToLower(candidate.SourceName) + ":" + strings.ToLower(candidate.ExternalID)
}

func normalizeCandidateURL(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed.Fragment = ""
	values := parsed.Query()
	for key := range values {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "yclid" || lower == "fbclid" || lower == "gclid" {
			values.Del(key)
		}
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

func stableCandidateID(value string) string {
	sum := sha1.Sum([]byte(value))
	return "candidate-" + hex.EncodeToString(sum[:8])
}

func cleanCandidateTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, tag)
	}
	return cleaned
}

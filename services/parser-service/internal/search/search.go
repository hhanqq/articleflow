package search

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type Parser interface {
	SourceName() string
	Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error)
}

type StrategyReporter interface {
	Strategy() string
}

type Usecase struct {
	producer      articleflowkafka.Producer
	parsers       map[string]Parser
	sourceOrder   []string
	sourceEnabled func(string) bool
}

func NewUsecase(producer articleflowkafka.Producer, parsers []Parser) *Usecase {
	return NewUsecaseWithSourceEnabled(producer, parsers, nil)
}

func NewUsecaseWithSourceEnabled(producer articleflowkafka.Producer, parsers []Parser, sourceEnabled func(string) bool) *Usecase {
	bySource := make(map[string]Parser, len(parsers))
	sourceOrder := make([]string, 0, len(parsers))
	for _, parser := range parsers {
		source := strings.TrimSpace(parser.SourceName())
		if source == "" {
			continue
		}
		bySource[source] = parser
		sourceOrder = append(sourceOrder, source)
	}
	if sourceEnabled == nil {
		sourceEnabled = func(string) bool { return true }
	}
	return &Usecase{producer: producer, parsers: bySource, sourceOrder: sourceOrder, sourceEnabled: sourceEnabled}
}

func (usecase *Usecase) SearchAndPublish(ctx context.Context, query parserv1.SearchQuery) (parserv1.SearchResult, error) {
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return parserv1.SearchResult{}, err
	}

	var buckets []sourceCandidateBucket
	var stats []parserv1.SourceStats
	seenCandidates := make(map[string]bool)
	var lastErr error
	for _, source := range usecase.selectedSources(query) {
		startedAt := time.Now()
		parser := usecase.parsers[source]
		candidates, err := parser.Search(ctx, query)
		sourceStats := parserv1.SourceStats{
			SourceName: source,
			Strategy:   sourceStrategy(parser),
			Status:     parserv1.SourceStatusOK,
			DurationMS: time.Since(startedAt).Milliseconds(),
		}
		if err != nil {
			sourceStats.Status = parserv1.SourceStatusFailed
			sourceStats.Error = err.Error()
			stats = append(stats, sourceStats)
			_ = usecase.publishFailure(ctx, source, query, err)
			lastErr = err
			continue
		}
		sourceStats.FoundCount = len(candidates)
		bucket := sourceCandidateBucket{source: source}
		var zeroScoreCandidates []scoredCandidate
		for _, candidate := range candidates {
			candidate = normalizeCandidate(candidate, source, query)
			if candidate.URL == "" || candidate.Title == "" {
				sourceStats.FilteredCount++
				continue
			}
			if !candidateWithinDateRange(candidate, query) {
				sourceStats.FilteredCount++
				continue
			}
			keys := candidateDedupKeys(candidate)
			if hasSeenCandidate(seenCandidates, keys) {
				sourceStats.FilteredCount++
				continue
			}
			markSeenCandidate(seenCandidates, keys)
			score := candidateRelevanceScore(candidate, query.Text)
			if score <= 0 {
				zeroScoreCandidates = append(zeroScoreCandidates, scoredCandidate{candidate: candidate, score: score})
				continue
			}
			bucket.scoredCandidates = append(bucket.scoredCandidates, scoredCandidate{candidate: candidate, score: score})
		}
		if len(bucket.scoredCandidates) == 0 && len(zeroScoreCandidates) > 0 {
			bucket.scoredCandidates = append(bucket.scoredCandidates, zeroScoreCandidates...)
		} else {
			sourceStats.FilteredCount += len(zeroScoreCandidates)
		}
		sortScoredCandidates(bucket.scoredCandidates)
		sourceStats.AcceptedCount = len(bucket.scoredCandidates)
		if sourceStats.FoundCount == 0 || sourceStats.AcceptedCount == 0 {
			sourceStats.Status = parserv1.SourceStatusEmpty
		}
		if len(bucket.scoredCandidates) > 0 {
			buckets = append(buckets, bucket)
		}
		stats = append(stats, sourceStats)
	}
	allCandidates := interleaveCandidateBuckets(buckets, query.Limit)
	returnedBySource := countCandidatesBySource(allCandidates)
	for index := range stats {
		stats[index].ReturnedCount = returnedBySource[stats[index].SourceName]
	}
	result := parserv1.SearchResult{Candidates: allCandidates, SourceStats: stats}
	if len(allCandidates) == 0 && lastErr != nil {
		return result, lastErr
	}
	for _, candidate := range allCandidates {
		if err := usecase.publishCandidate(ctx, candidate); err != nil {
			return result, err
		}
	}
	publishedBySource := countCandidatesBySource(allCandidates)
	for index := range result.SourceStats {
		result.SourceStats[index].PublishedCount = publishedBySource[result.SourceStats[index].SourceName]
	}
	return result, nil
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

func (usecase *Usecase) selectedSources(query parserv1.SearchQuery) []string {
	if len(query.Sources) > 0 {
		sources := make([]string, 0, len(query.Sources))
		for _, source := range query.Sources {
			source = strings.TrimSpace(source)
			if source == "" || usecase.parsers[source] == nil || !usecase.sourceEnabled(source) {
				continue
			}
			sources = append(sources, source)
		}
		return sources
	}
	sources := make([]string, 0, len(usecase.sourceOrder))
	for _, source := range usecase.sourceOrder {
		if usecase.sourceEnabled(source) {
			sources = append(sources, source)
		}
	}
	return sources
}

func candidateWithinDateRange(candidate parserv1.ArticleCandidate, query parserv1.SearchQuery) bool {
	if candidate.PublishedAt.IsZero() {
		return true
	}
	if query.FromDate != nil && candidate.PublishedAt.Before(query.FromDate.UTC()) {
		return false
	}
	if query.ToDate != nil && candidate.PublishedAt.After(query.ToDate.UTC()) {
		return false
	}
	return true
}

func sourceStrategy(parser Parser) string {
	reporter, ok := parser.(StrategyReporter)
	if !ok {
		return "unknown"
	}
	strategy := strings.TrimSpace(reporter.Strategy())
	if strategy == "" {
		return "unknown"
	}
	return strategy
}

type sourceCandidateBucket struct {
	source           string
	scoredCandidates []scoredCandidate
}

type scoredCandidate struct {
	candidate parserv1.ArticleCandidate
	score     int
}

func interleaveCandidateBuckets(buckets []sourceCandidateBucket, limit int) []parserv1.ArticleCandidate {
	if len(buckets) == 0 || limit <= 0 {
		return nil
	}
	result := make([]parserv1.ArticleCandidate, 0, limit)
	for index := 0; len(result) < limit; index++ {
		added := false
		for _, bucket := range buckets {
			if index >= len(bucket.scoredCandidates) {
				continue
			}
			result = append(result, bucket.scoredCandidates[index].candidate)
			added = true
			if len(result) >= limit {
				break
			}
		}
		if !added {
			break
		}
	}
	return result
}

func sortScoredCandidates(candidates []scoredCandidate) {
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].score != candidates[right].score {
			return candidates[left].score > candidates[right].score
		}
		return candidates[left].candidate.PublishedAt.After(candidates[right].candidate.PublishedAt)
	})
}

func countCandidatesBySource(candidates []parserv1.ArticleCandidate) map[string]int {
	counts := make(map[string]int)
	for _, candidate := range candidates {
		counts[candidate.SourceName]++
	}
	return counts
}

func candidateRelevanceScore(candidate parserv1.ArticleCandidate, queryText string) int {
	terms := queryTerms(queryText)
	if len(terms) == 0 {
		return 1
	}
	title := strings.ToLower(candidate.Title)
	summary := strings.ToLower(candidate.Summary)
	content := strings.ToLower(candidate.Content)
	urlText := strings.ToLower(candidate.URL)
	tagText := strings.ToLower(strings.Join(candidate.Tags, " "))

	score := 0
	for _, term := range terms {
		if strings.Contains(title, term) {
			score += 4
		}
		if strings.Contains(tagText, term) {
			score += 3
		}
		if strings.Contains(summary, term) {
			score += 2
		}
		if strings.Contains(content, term) {
			score += 1
		}
		if strings.Contains(urlText, term) {
			score += 1
		}
	}
	return score
}

func queryTerms(queryText string) []string {
	rawTerms := strings.FieldsFunc(strings.ToLower(queryText), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	terms := make([]string, 0, len(rawTerms))
	seen := make(map[string]bool, len(rawTerms))
	for _, term := range rawTerms {
		term = strings.TrimSpace(term)
		if len([]rune(term)) < 2 || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
	}
	return terms
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

func candidateDedupKeys(candidate parserv1.ArticleCandidate) []string {
	keys := make([]string, 0, 3)
	if candidate.URL != "" {
		keys = append(keys, "url:"+strings.ToLower(candidate.URL))
	}
	if candidate.SourceName != "" && candidate.ExternalID != "" {
		keys = append(keys, "source:"+strings.ToLower(candidate.SourceName)+":"+strings.ToLower(candidate.ExternalID))
	}
	if titleKey := candidateTitleDedupKey(candidate.Title); titleKey != "" {
		keys = append(keys, "title:"+titleKey)
	}
	return keys
}

func hasSeenCandidate(seen map[string]bool, keys []string) bool {
	for _, key := range keys {
		if seen[key] {
			return true
		}
	}
	return false
}

func markSeenCandidate(seen map[string]bool, keys []string) {
	for _, key := range keys {
		seen[key] = true
	}
}

func normalizeCandidateURL(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = canonicalHost(strings.ToLower(parsed.Host))
	parsed.Fragment = ""
	values := parsed.Query()
	for key := range values {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "yclid" || lower == "fbclid" || lower == "gclid" {
			values.Del(key)
		}
	}
	parsed.RawQuery = values.Encode()
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String()
}

func canonicalHost(host string) string {
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	return host
}

func candidateTitleDedupKey(title string) string {
	terms := strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if len([]rune(term)) < 3 {
			continue
		}
		normalized = append(normalized, term)
	}
	if len(normalized) < 4 {
		return ""
	}
	return strings.Join(normalized, " ")
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

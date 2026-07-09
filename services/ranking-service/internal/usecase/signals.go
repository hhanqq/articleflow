package usecase

import (
	"fmt"
	"strings"
	"sync"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type SignalStore struct {
	mu             sync.RWMutex
	articleTags    map[string][]string
	articleSources map[string]string
	tagWeights     map[string]float64
	sourceWeights  map[string]float64
}

func NewSignalStore() *SignalStore {
	return &SignalStore{
		articleTags:    make(map[string][]string),
		articleSources: make(map[string]string),
		tagWeights:     make(map[string]float64),
		sourceWeights:  make(map[string]float64),
	}
}

func (store *SignalStore) RememberArticle(item feedv1.FeedItem) {
	if strings.TrimSpace(item.ArticleID) == "" {
		return
	}
	tags := normalizedTags(item)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(tags) > 0 {
		store.articleTags[item.ArticleID] = tags
	}
	if source := normalizedSource(item.SourceName); source != "" {
		store.articleSources[item.ArticleID] = source
	}
}

func (store *SignalStore) RecordReaction(event eventsv1.UserReactionCreatedEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	weight, err := reactionWeight(userv1.ReactionType(event.Type))
	if err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	for _, tag := range store.articleTags[event.ArticleID] {
		store.tagWeights[tag] += weight
	}
	if source := store.articleSources[event.ArticleID]; source != "" {
		store.sourceWeights[source] += weight / 2
	}
	return nil
}

func (store *SignalStore) ApplyToScore(baseScore float64, item feedv1.FeedItem) float64 {
	tags := normalizedTags(item)
	source := normalizedSource(item.SourceName)
	if len(tags) == 0 && source == "" {
		return baseScore
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	score := baseScore
	for _, tag := range tags {
		score += store.tagWeights[tag]
	}
	if source != "" {
		score += store.sourceWeights[source]
	}
	if score < 0 {
		return 0
	}
	return score
}

func normalizedTags(item feedv1.FeedItem) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		normalized := strings.TrimSpace(strings.ToLower(tag))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		tags = append(tags, normalized)
	}
	return tags
}

func reactionWeight(reactionType userv1.ReactionType) (float64, error) {
	switch reactionType {
	case userv1.ReactionLike:
		return 4, nil
	case userv1.ReactionSave:
		return 8, nil
	case userv1.ReactionOpen:
		return 1, nil
	case userv1.ReactionSkip:
		return -3, nil
	case userv1.ReactionDislike:
		return -8, nil
	default:
		return 0, fmt.Errorf("unsupported reaction type: %s", reactionType)
	}
}

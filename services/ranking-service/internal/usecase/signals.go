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
	mu          sync.RWMutex
	articleTags map[string][]string
	tagWeights  map[string]float64
}

func NewSignalStore() *SignalStore {
	return &SignalStore{
		articleTags: make(map[string][]string),
		tagWeights:  make(map[string]float64),
	}
}

func (store *SignalStore) RememberArticle(item feedv1.FeedItem) {
	if strings.TrimSpace(item.ArticleID) == "" {
		return
	}
	tags := normalizedTags(item)
	if len(tags) == 0 {
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.articleTags[item.ArticleID] = tags
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
	return nil
}

func (store *SignalStore) ApplyToScore(baseScore float64, item feedv1.FeedItem) float64 {
	tags := normalizedTags(item)
	if len(tags) == 0 {
		return baseScore
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	score := baseScore
	for _, tag := range tags {
		score += store.tagWeights[tag]
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

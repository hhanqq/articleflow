package usecase

import (
	"fmt"
	"strings"
	"sync"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type ReactionStore interface {
	Record(reaction userv1.UserReaction) error
}

type MemoryReactions struct {
	mu        sync.RWMutex
	reactions map[string]userv1.UserReaction
}

func NewMemoryReactions() *MemoryReactions {
	return &MemoryReactions{
		reactions: make(map[string]userv1.UserReaction),
	}
}

func (store *MemoryReactions) Record(reaction userv1.UserReaction) error {
	if strings.TrimSpace(reaction.UserID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(reaction.ArticleID) == "" {
		return fmt.Errorf("article id is required")
	}
	if strings.TrimSpace(string(reaction.Type)) == "" {
		return fmt.Errorf("reaction type is required")
	}
	if reaction.CreatedAt.IsZero() {
		return fmt.Errorf("created at is required")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.reactions[reactionKey(reaction)] = reaction
	return nil
}

func (store *MemoryReactions) ListByUser(userID string) []userv1.UserReaction {
	store.mu.RLock()
	defer store.mu.RUnlock()

	reactions := make([]userv1.UserReaction, 0)
	for _, reaction := range store.reactions {
		if reaction.UserID == userID {
			reactions = append(reactions, reaction)
		}
	}
	return reactions
}

func reactionKey(reaction userv1.UserReaction) string {
	return reaction.UserID + ":" + reaction.ArticleID + ":" + string(reaction.Type)
}

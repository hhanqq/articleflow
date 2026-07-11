package usecase

import (
	"fmt"
	"strings"
	"sync"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type ReactionStore interface {
	Record(reaction userv1.UserReaction) error
	ListByUser(userID string) []userv1.UserReaction
	EnsureProfile(profile userv1.UserProfile) (userv1.UserProfile, error)
	GetProfile(userID string) (userv1.UserProfile, bool)
}

type MemoryReactions struct {
	mu        sync.RWMutex
	reactions map[string]userv1.UserReaction
	profiles  map[string]userv1.UserProfile
}

func NewMemoryReactions() *MemoryReactions {
	return &MemoryReactions{
		reactions: make(map[string]userv1.UserReaction),
		profiles:  make(map[string]userv1.UserProfile),
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
	if _, err := store.EnsureProfile(userv1.UserProfile{
		ID:        reaction.UserID,
		CreatedAt: reaction.CreatedAt,
	}); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.reactions[reactionKey(reaction)] = reaction
	return nil
}

func (store *MemoryReactions) EnsureProfile(profile userv1.UserProfile) (userv1.UserProfile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	if profile.ID == "" {
		return userv1.UserProfile{}, fmt.Errorf("user id is required")
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now().UTC()
	}
	profile.Email = strings.TrimSpace(profile.Email)
	profile.Interests = append([]string(nil), profile.Interests...)

	store.mu.Lock()
	defer store.mu.Unlock()
	if existing, ok := store.profiles[profile.ID]; ok {
		if profile.Email != "" {
			existing.Email = profile.Email
		}
		if len(profile.Interests) > 0 {
			existing.Interests = append([]string(nil), profile.Interests...)
		}
		store.profiles[profile.ID] = existing
		return existing, nil
	}
	store.profiles[profile.ID] = profile
	return profile, nil
}

func (store *MemoryReactions) GetProfile(userID string) (userv1.UserProfile, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	profile, ok := store.profiles[strings.TrimSpace(userID)]
	if !ok {
		return userv1.UserProfile{}, false
	}
	profile.Interests = append([]string(nil), profile.Interests...)
	return profile, true
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

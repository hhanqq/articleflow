package userv1

import "time"

type ReactionType string

const (
	ReactionLike    ReactionType = "like"
	ReactionDislike ReactionType = "dislike"
	ReactionSkip    ReactionType = "skip"
	ReactionSave    ReactionType = "save"
	ReactionOpen    ReactionType = "open"
)

type UserReaction struct {
	UserID    string
	ArticleID string
	Type      ReactionType
	CreatedAt time.Time
}

type UserProfile struct {
	ID        string
	Email     string
	Interests []string
	CreatedAt time.Time
}


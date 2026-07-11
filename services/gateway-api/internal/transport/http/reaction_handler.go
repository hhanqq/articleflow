package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type ReactionRecorder interface {
	Record(reaction userv1.UserReaction) error
}

type ReactionRequest struct {
	UserID    string `json:"user_id"`
	ArticleID string `json:"article_id"`
	Type      string `json:"type"`
}

type ReactionResponse struct {
	Accepted bool `json:"accepted"`
}

func NewReactionHandler(recorder ReactionRecorder) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload ReactionRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "invalid json body", http.StatusBadRequest)
			return
		}
		reaction, err := payload.toReaction(time.Now().UTC(), UserIDFromRequest(request))
		if err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		if err := recorder.Record(reaction); err != nil {
			http.Error(response, "reaction failed", http.StatusBadGateway)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(response).Encode(ReactionResponse{Accepted: true})
	})
}

func (request ReactionRequest) toReaction(now time.Time, fallbackUserID string) (userv1.UserReaction, error) {
	reaction := userv1.UserReaction{
		UserID:    strings.TrimSpace(request.UserID),
		ArticleID: strings.TrimSpace(request.ArticleID),
		Type:      userv1.ReactionType(strings.TrimSpace(request.Type)),
		CreatedAt: now,
	}
	if reaction.UserID == "" {
		reaction.UserID = strings.TrimSpace(fallbackUserID)
	}
	if reaction.UserID == "" {
		return userv1.UserReaction{}, errors.New("user_id is required")
	}
	if reaction.ArticleID == "" {
		return userv1.UserReaction{}, errors.New("article_id is required")
	}
	if !isAllowedReactionType(reaction.Type) {
		return userv1.UserReaction{}, errors.New("unsupported reaction type")
	}
	return reaction, nil
}

func isAllowedReactionType(reactionType userv1.ReactionType) bool {
	switch reactionType {
	case userv1.ReactionLike, userv1.ReactionDislike, userv1.ReactionSkip, userv1.ReactionSave, userv1.ReactionOpen:
		return true
	default:
		return false
	}
}

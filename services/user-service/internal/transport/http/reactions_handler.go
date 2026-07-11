package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type ReactionReader interface {
	ListByUser(userID string) []userv1.UserReaction
}

type UserReactionsResponse struct {
	Reactions []userv1.UserReaction `json:"reactions"`
}

func NewUserReactionsHandler(reader ReactionReader) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := userIDFromPath(request.URL.Path)
		if userID == "" {
			http.Error(response, "user id is required", http.StatusBadRequest)
			return
		}
		reactions := reader.ListByUser(userID)
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(UserReactionsResponse{Reactions: reactions})
	})
}

func userIDFromPath(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/users/")
	userID, _, _ := strings.Cut(path, "/reactions")
	return strings.TrimSpace(userID)
}

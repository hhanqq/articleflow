package httptransport

import (
	"context"
	"encoding/json"
	"net/http"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type UserProfileProvider interface {
	EnsureUserProfile(ctx context.Context, userID string) (userv1.UserProfile, error)
}

type MeResponse struct {
	User userv1.UserProfile `json:"user"`
}

func NewMeHandler(provider UserProfileProvider) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromRequest(request)
		if userID == "" {
			http.Error(response, "user session is required", http.StatusUnauthorized)
			return
		}
		profile := userv1.UserProfile{ID: userID}
		if provider != nil {
			created, err := provider.EnsureUserProfile(request.Context(), userID)
			if err != nil {
				http.Error(response, "user profile failed", http.StatusBadGateway)
				return
			}
			profile = created
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(MeResponse{User: profile})
	})
}

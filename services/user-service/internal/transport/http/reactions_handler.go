package httptransport

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type ReactionReader interface {
	ListByUser(userID string) []userv1.UserReaction
}

type ProfileEnsurer interface {
	EnsureProfile(profile userv1.UserProfile) (userv1.UserProfile, error)
}

type UserStore interface {
	ReactionReader
	ProfileEnsurer
}

type UserReactionsResponse struct {
	Reactions []userv1.UserReaction `json:"reactions"`
}

type UserProfileResponse struct {
	User userv1.UserProfile `json:"user"`
}

type userProfileRequest struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Interests []string `json:"interests"`
}

func NewUsersHandler(store UserStore) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/v1/users" {
			ensureUserProfile(response, request, store)
			return
		}
		if strings.HasSuffix(request.URL.Path, "/reactions") {
			NewUserReactionsHandler(store).ServeHTTP(response, request)
			return
		}
		http.NotFound(response, request)
	})
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

func ensureUserProfile(response http.ResponseWriter, request *http.Request, store ProfileEnsurer) {
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload userProfileRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(response, "invalid json body", http.StatusBadRequest)
		return
	}
	profile, err := store.EnsureProfile(userv1.UserProfile{
		ID:        strings.TrimSpace(payload.ID),
		Email:     strings.TrimSpace(payload.Email),
		Interests: append([]string(nil), payload.Interests...),
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		http.Error(response, "user profile failed", http.StatusBadRequest)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(UserProfileResponse{User: profile})
}

func userIDFromPath(path string) string {
	path = strings.TrimPrefix(path, "/api/v1/users/")
	userID, _, _ := strings.Cut(path, "/reactions")
	return strings.TrimSpace(userID)
}

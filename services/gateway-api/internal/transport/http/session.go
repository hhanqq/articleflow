package httptransport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const UserIDCookieName = "articleflow_user_id"

type userIDContextKey struct{}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, strings.TrimSpace(userID))
}

func UserIDFromRequest(request *http.Request) string {
	if request == nil {
		return ""
	}
	if userID, ok := request.Context().Value(userIDContextKey{}).(string); ok && strings.TrimSpace(userID) != "" {
		return strings.TrimSpace(userID)
	}
	if userID := strings.TrimSpace(request.Header.Get("X-Articleflow-User-ID")); userID != "" {
		return userID
	}
	if userID := strings.TrimSpace(request.URL.Query().Get("user_id")); userID != "" {
		return userID
	}
	return ""
}

func NewSessionMiddleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userID := strings.TrimSpace(request.Header.Get("X-Articleflow-User-ID"))
		if userID == "" {
			if cookie, err := request.Cookie(UserIDCookieName); err == nil {
				userID = strings.TrimSpace(cookie.Value)
			}
		}
		if userID == "" {
			userID = generateAnonymousUserID()
		}
		http.SetCookie(response, &http.Cookie{
			Name:     UserIDCookieName,
			Value:    userID,
			Path:     "/",
			MaxAge:   60 * 60 * 24 * 365,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		next.ServeHTTP(response, request.WithContext(ContextWithUserID(request.Context(), userID)))
	})
}

func generateAnonymousUserID() string {
	var payload [16]byte
	if _, err := rand.Read(payload[:]); err != nil {
		return "anon_local"
	}
	return "anon_" + hex.EncodeToString(payload[:])
}

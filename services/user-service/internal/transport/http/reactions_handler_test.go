package httptransport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type fakeReactionReader struct {
	reactions []userv1.UserReaction
	userID    string
}

func (reader *fakeReactionReader) ListByUser(userID string) []userv1.UserReaction {
	reader.userID = userID
	return reader.reactions
}

func TestUserReactionsHandlerListsReactionsForUser(t *testing.T) {
	reader := &fakeReactionReader{
		reactions: []userv1.UserReaction{
			{
				UserID:    "reader-1",
				ArticleID: "habr:1",
				Type:      userv1.ReactionSave,
				CreatedAt: time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewUserReactionsHandler(reader)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/reader-1/reactions", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if reader.userID != "reader-1" {
		t.Fatalf("expected reader-1 lookup, got %q", reader.userID)
	}
	var payload UserReactionsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Reactions) != 1 || payload.Reactions[0].Type != userv1.ReactionSave {
		t.Fatalf("unexpected reactions response: %#v", payload.Reactions)
	}
}

package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type fakeReactionRecorder struct {
	reaction userv1.UserReaction
}

func (recorder *fakeReactionRecorder) Record(reaction userv1.UserReaction) error {
	recorder.reaction = reaction
	return nil
}

func TestReactionHandlerAcceptsReaction(t *testing.T) {
	recorder := &fakeReactionRecorder{}
	handler := NewReactionHandler(recorder)
	body := bytes.NewBufferString(`{"user_id":"user-1","article_id":"article-1","type":"save"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reactions", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", response.Code)
	}
	var payload ReactionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Accepted {
		t.Fatal("expected accepted response")
	}
	if recorder.reaction.Type != userv1.ReactionSave {
		t.Fatalf("unexpected reaction type: %s", recorder.reaction.Type)
	}
}

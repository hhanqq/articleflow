package user

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListsUserReactions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/users/reader-1/reactions" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"reactions":[{"UserID":"reader-1","ArticleID":"habr:1","Type":"save","CreatedAt":"2026-07-11T10:00:00Z"}]}`))
	}))
	defer server.Close()
	client := New(server.URL, server.Client())

	reactions, err := client.ListUserReactions(t.Context(), "reader-1")

	if err != nil {
		t.Fatalf("list reactions: %v", err)
	}
	if len(reactions) != 1 || reactions[0].ArticleID != "habr:1" {
		t.Fatalf("unexpected reactions: %#v", reactions)
	}
}

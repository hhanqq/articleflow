package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type reactionsResponse struct {
	Reactions []userv1.UserReaction `json:"reactions"`
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (client *Client) ListUserReactions(ctx context.Context, userID string) ([]userv1.UserReaction, error) {
	endpoint, err := url.Parse(client.baseURL + "/api/v1/users/" + url.PathEscape(userID) + "/reactions")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, nil
	}
	var payload reactionsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Reactions, nil
}

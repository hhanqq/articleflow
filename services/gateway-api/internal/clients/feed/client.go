package feed

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type responsePayload struct {
	Items []feedv1.FeedItem `json:"items"`
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

func (client *Client) List(limit int) []feedv1.FeedItem {
	endpoint, err := url.Parse(client.baseURL + "/api/v1/feed")
	if err != nil {
		return nil
	}
	query := endpoint.Query()
	query.Set("limit", fmt.Sprintf("%d", limit))
	endpoint.RawQuery = query.Encode()

	response, err := client.httpClient.Get(endpoint.String())
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil
	}

	var payload responsePayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil
	}
	return payload.Items
}

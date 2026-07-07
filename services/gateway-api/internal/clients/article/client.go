package article

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type responsePayload struct {
	Article articlev1.Article `json:"article"`
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

func (client *Client) GetByID(id string) (articlev1.Article, bool) {
	endpoint, err := url.Parse(client.baseURL + "/api/v1/articles")
	if err != nil {
		return articlev1.Article{}, false
	}
	query := endpoint.Query()
	query.Set("id", id)
	endpoint.RawQuery = query.Encode()

	response, err := client.httpClient.Get(endpoint.String())
	if err != nil {
		return articlev1.Article{}, false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return articlev1.Article{}, false
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return articlev1.Article{}, false
	}

	var payload responsePayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return articlev1.Article{}, false
	}
	return payload.Article, true
}

package article

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type responsePayload struct {
	Article articlev1.Article `json:"article"`
}

type searchRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

type searchResponse struct {
	Articles []articlev1.Article `json:"articles"`
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

func (client *Client) Search(query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	payload, err := json.Marshal(searchRequest{
		Query:   query.Text,
		Sources: query.Sources,
		Limit:   query.Limit,
	})
	if err != nil {
		return nil, err
	}
	response, err := client.httpClient.Post(client.baseURL+"/api/v1/articles/search", "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, errStatus(response.StatusCode)
	}
	var decoded searchResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	candidates := make([]parserv1.ArticleCandidate, 0, len(decoded.Articles))
	for _, article := range decoded.Articles {
		candidates = append(candidates, parserv1.ArticleCandidate{
			SourceName:  article.SourceName,
			ExternalID:  article.ExternalID,
			URL:         article.URL,
			Title:       article.Title,
			Summary:     article.Summary,
			Content:     article.Content,
			Author:      article.Author,
			Tags:        article.Tags,
			Language:    article.Language,
			PublishedAt: article.PublishedAt,
		})
	}
	return candidates, nil
}

type errStatus int

func (err errStatus) Error() string {
	return "article-service returned status " + http.StatusText(int(err))
}

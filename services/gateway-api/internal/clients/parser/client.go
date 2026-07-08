package parserclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type jobRequest struct {
	Query   string   `json:"query"`
	Sources []string `json:"sources"`
	Limit   int      `json:"limit"`
}

type jobResponse struct {
	Job parserv1.ParserJob `json:"job"`
}

type jobsResponse struct {
	Jobs []parserv1.ParserJob `json:"jobs"`
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (client *Client) StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	payload, err := json.Marshal(jobRequest{
		Query:   query.Text,
		Sources: query.Sources,
		Limit:   query.Limit,
	})
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/v1/parser/jobs", bytes.NewReader(payload))
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return parserv1.ParserJob{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parserv1.ParserJob{}, fmt.Errorf("parser-service returned status %d", response.StatusCode)
	}
	var decoded jobResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return parserv1.ParserJob{}, err
	}
	return decoded.Job, nil
}

func (client *Client) Get(ctx context.Context, id string) (parserv1.ParserJob, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/v1/parser/jobs/"+id, nil)
	if err != nil {
		return parserv1.ParserJob{}, false, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return parserv1.ParserJob{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return parserv1.ParserJob{}, false, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parserv1.ParserJob{}, false, fmt.Errorf("parser-service returned status %d", response.StatusCode)
	}
	var decoded jobResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return parserv1.ParserJob{}, false, err
	}
	return decoded.Job, true, nil
}

func (client *Client) List(ctx context.Context, limit int) ([]parserv1.ParserJob, error) {
	if limit <= 0 {
		limit = 20
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/v1/parser/jobs?limit="+strconv.Itoa(limit), nil)
	if err != nil {
		return nil, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("parser-service returned status %d", response.StatusCode)
	}
	var decoded jobsResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded.Jobs, nil
}

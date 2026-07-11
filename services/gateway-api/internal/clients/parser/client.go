package parserclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type jobRequest struct {
	Query    string     `json:"query"`
	Sources  []string   `json:"sources"`
	Limit    int        `json:"limit"`
	FromDate *time.Time `json:"from_date,omitempty"`
	ToDate   *time.Time `json:"to_date,omitempty"`
}

type jobResponse struct {
	Job parserv1.ParserJob `json:"job"`
}

type jobsResponse struct {
	Jobs []parserv1.ParserJob `json:"jobs"`
}

type sourcesResponse struct {
	Sources []parserv1.ParserSource `json:"sources"`
}

type sourceResponse struct {
	Source parserv1.ParserSource `json:"source"`
}

type sourceUpdateRequest struct {
	Enabled bool `json:"enabled"`
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
		Query:    query.Text,
		Sources:  query.Sources,
		Limit:    query.Limit,
		FromDate: query.FromDate,
		ToDate:   query.ToDate,
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

func (client *Client) ListSources(ctx context.Context) ([]parserv1.ParserSource, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/v1/parser/sources", nil)
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
	var decoded sourcesResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded.Sources, nil
}

func (client *Client) SetSourceEnabled(ctx context.Context, name string, enabled bool) (parserv1.ParserSource, bool, error) {
	payload, err := json.Marshal(sourceUpdateRequest{Enabled: enabled})
	if err != nil {
		return parserv1.ParserSource{}, false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, client.baseURL+"/api/v1/parser/sources/"+strings.TrimSpace(name), bytes.NewReader(payload))
	if err != nil {
		return parserv1.ParserSource{}, false, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return parserv1.ParserSource{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return parserv1.ParserSource{}, false, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parserv1.ParserSource{}, false, fmt.Errorf("parser-service returned status %d", response.StatusCode)
	}
	var decoded sourceResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return parserv1.ParserSource{}, false, err
	}
	return decoded.Source, true, nil
}

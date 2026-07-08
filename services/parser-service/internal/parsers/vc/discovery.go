package vc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultDiscoveryEndpoint = "https://api.vc.ru/v2.10/search/posts"

type DiscoverySearcherOptions struct {
	Endpoint string
	Client   *http.Client
}

type DiscoverySearcher struct {
	endpoint string
	client   *http.Client
}

func NewDiscoverySearcher(options DiscoverySearcherOptions) *DiscoverySearcher {
	httpClient := options.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &DiscoverySearcher{
		endpoint: firstNonEmpty(options.Endpoint, defaultDiscoveryEndpoint),
		client:   httpClient,
	}
}

func (searcher *DiscoverySearcher) SearchURLs(ctx context.Context, query string, limit int) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	requestURL, err := searcher.searchURL(query)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "articleflow-parser/0.1")

	response, err := searcher.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
	}

	var payload discoveryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return discoveryURLs(payload, limit), nil
}

func (searcher *DiscoverySearcher) searchURL(query string) (string, error) {
	parsed, err := url.Parse(searcher.endpoint)
	if err != nil {
		return "", err
	}
	values := parsed.Query()
	values.Set("markdown", "false")
	values.Set("q", query)
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

type discoveryResponse struct {
	Result struct {
		Items []struct {
			Type string `json:"type"`
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"items"`
	} `json:"result"`
}

func discoveryURLs(payload discoveryResponse, limit int) []string {
	urls := make([]string, 0, len(payload.Result.Items))
	seen := make(map[string]bool, len(payload.Result.Items))
	for _, item := range payload.Result.Items {
		if item.Type != "" && item.Type != "entry" {
			continue
		}
		cleaned := cleanURL(item.Data.URL)
		if cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		urls = append(urls, cleaned)
		if limit > 0 && len(urls) >= limit {
			break
		}
	}
	return urls
}

package vc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultDiscoveryEndpoint = "https://api.vc.ru/v2.10/search/posts"
const defaultDiscoveryLimit = 20
const maxDiscoveryLimit = 100

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
	limit = normalizeDiscoveryLimit(limit)
	var page discoveryPage
	urls := make([]string, 0, limit)
	seen := make(map[string]bool, limit)
	for len(urls) < limit {
		payload, err := searcher.searchPage(ctx, query, page)
		if err != nil {
			return nil, err
		}
		added := appendDiscoveryURLs(urls, seen, payload, limit)
		if len(added) == len(urls) {
			return added, nil
		}
		urls = added
		nextPage, ok := nextDiscoveryPage(payload)
		if !ok {
			return urls, nil
		}
		page = nextPage
	}
	return urls, nil
}

func (searcher *DiscoverySearcher) searchPage(ctx context.Context, query string, page discoveryPage) (discoveryResponse, error) {
	requestURL, err := searcher.searchURL(query, page)
	if err != nil {
		return discoveryResponse{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return discoveryResponse{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "articleflow-parser/0.1")

	response, err := searcher.client.Do(request)
	if err != nil {
		return discoveryResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return discoveryResponse{}, fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
	}

	var payload discoveryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return discoveryResponse{}, err
	}
	return payload, nil
}

func (searcher *DiscoverySearcher) searchURL(query string, page discoveryPage) (string, error) {
	parsed, err := url.Parse(searcher.endpoint)
	if err != nil {
		return "", err
	}
	values := parsed.Query()
	values.Set("markdown", "false")
	values.Set("q", query)
	if page.LastID > 0 {
		values.Set("lastId", strconv.FormatInt(page.LastID, 10))
	}
	if page.LastSortingValue > 0 {
		values.Set("lastSortingValue", strconv.FormatInt(page.LastSortingValue, 10))
	}
	if page.Cursor != "" {
		values.Set("cursor", page.Cursor)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

type discoveryResponse struct {
	Result struct {
		Cursor           string `json:"cursor"`
		LastID           int64  `json:"lastId"`
		LastSortingValue int64  `json:"lastSortingValue"`
		Items            []struct {
			Type string `json:"type"`
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"items"`
	} `json:"result"`
}

type discoveryPage struct {
	Cursor           string
	LastID           int64
	LastSortingValue int64
}

func appendDiscoveryURLs(urls []string, seen map[string]bool, payload discoveryResponse, limit int) []string {
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

func nextDiscoveryPage(payload discoveryResponse) (discoveryPage, bool) {
	page := discoveryPage{
		Cursor:           strings.TrimSpace(payload.Result.Cursor),
		LastID:           payload.Result.LastID,
		LastSortingValue: payload.Result.LastSortingValue,
	}
	if page.Cursor == "" && (page.LastID <= 0 || page.LastSortingValue <= 0) {
		return discoveryPage{}, false
	}
	return page, true
}

func normalizeDiscoveryLimit(limit int) int {
	if limit <= 0 {
		return defaultDiscoveryLimit
	}
	if limit > maxDiscoveryLimit {
		return maxDiscoveryLimit
	}
	return limit
}

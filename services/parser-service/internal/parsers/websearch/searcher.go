package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGoogleEndpoint = "https://www.googleapis.com/customsearch/v1"
	defaultBingEndpoint   = "https://api.bing.microsoft.com/v7.0/search"
	defaultSite           = "vc.ru"
)

type GoogleOptions struct {
	Endpoint string
	APIKey   string
	CX       string
	Site     string
	Client   *http.Client
}

type GoogleSearcher struct {
	endpoint string
	apiKey   string
	cx       string
	site     string
	client   *http.Client
}

func NewGoogleSearcher(options GoogleOptions) *GoogleSearcher {
	return &GoogleSearcher{
		endpoint: firstNonEmpty(options.Endpoint, defaultGoogleEndpoint),
		apiKey:   strings.TrimSpace(options.APIKey),
		cx:       strings.TrimSpace(options.CX),
		site:     firstNonEmpty(options.Site, defaultSite),
		client:   httpClient(options.Client),
	}
}

func (searcher *GoogleSearcher) SearchURLs(ctx context.Context, query string, limit int) ([]string, error) {
	if searcher.apiKey == "" || searcher.cx == "" {
		return nil, fmt.Errorf("google search api key and cx are required")
	}
	values := url.Values{}
	values.Set("key", searcher.apiKey)
	values.Set("cx", searcher.cx)
	values.Set("q", siteQuery(searcher.site, query))
	values.Set("num", fmt.Sprint(normalizeLimit(limit)))

	var payload googleResponse
	if err := searcher.getJSON(ctx, searcher.endpoint+"?"+values.Encode(), nil, &payload); err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		urls = append(urls, item.Link)
	}
	return filterSiteURLs(urls, searcher.site), nil
}

func (searcher *GoogleSearcher) getJSON(ctx context.Context, requestURL string, headers map[string]string, target any) error {
	return getJSON(ctx, searcher.client, requestURL, headers, target)
}

type googleResponse struct {
	Items []struct {
		Link string `json:"link"`
	} `json:"items"`
}

type BingOptions struct {
	Endpoint string
	APIKey   string
	Site     string
	Client   *http.Client
}

type BingSearcher struct {
	endpoint string
	apiKey   string
	site     string
	client   *http.Client
}

func NewBingSearcher(options BingOptions) *BingSearcher {
	return &BingSearcher{
		endpoint: firstNonEmpty(options.Endpoint, defaultBingEndpoint),
		apiKey:   strings.TrimSpace(options.APIKey),
		site:     firstNonEmpty(options.Site, defaultSite),
		client:   httpClient(options.Client),
	}
}

func (searcher *BingSearcher) SearchURLs(ctx context.Context, query string, limit int) ([]string, error) {
	if searcher.apiKey == "" {
		return nil, fmt.Errorf("bing search api key is required")
	}
	values := url.Values{}
	values.Set("q", siteQuery(searcher.site, query))
	values.Set("count", fmt.Sprint(normalizeLimit(limit)))
	values.Set("responseFilter", "Webpages")

	var payload bingResponse
	headers := map[string]string{"Ocp-Apim-Subscription-Key": searcher.apiKey}
	if err := getJSON(ctx, searcher.client, searcher.endpoint+"?"+values.Encode(), headers, &payload); err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(payload.WebPages.Value))
	for _, item := range payload.WebPages.Value {
		urls = append(urls, item.URL)
	}
	return filterSiteURLs(urls, searcher.site), nil
}

type bingResponse struct {
	WebPages struct {
		Value []struct {
			URL string `json:"url"`
		} `json:"value"`
	} `json:"webPages"`
}

func getJSON(ctx context.Context, client *http.Client, requestURL string, headers map[string]string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "articleflow-parser/0.1")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("GET %s returned status %d", requestURL, response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func siteQuery(site string, query string) string {
	return "site:" + strings.TrimSpace(site) + " " + strings.TrimSpace(query)
}

func filterSiteURLs(values []string, site string) []string {
	site = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(site)), "www.")
	filtered := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		parsed, err := url.Parse(strings.TrimSpace(value))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		host := strings.TrimPrefix(strings.ToLower(parsed.Host), "www.")
		if host != site {
			continue
		}
		parsed.RawQuery = ""
		parsed.Fragment = ""
		cleaned := parsed.String()
		if cleaned == "" || seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		filtered = append(filtered, cleaned)
	}
	return filtered
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 10 {
		return 10
	}
	return limit
}

func httpClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

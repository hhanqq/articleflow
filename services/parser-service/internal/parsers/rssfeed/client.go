package rssfeed

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type ClientOptions struct {
	SourceName string
	FeedURL    string
	HTTPClient *http.Client
	Language   string
}

type Client struct {
	sourceName string
	feedURL    string
	httpClient *http.Client
	language   string
}

func NewClient(options ClientOptions) *Client {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	language := strings.TrimSpace(options.Language)
	if language == "" {
		language = "ru"
	}
	return &Client{
		sourceName: strings.TrimSpace(options.SourceName),
		feedURL:    strings.TrimSpace(options.FeedURL),
		httpClient: httpClient,
		language:   language,
	}
}

func (client *Client) SourceName() string {
	return client.sourceName
}

func (client *Client) Strategy() string {
	return "rss"
}

func (client *Client) Search(ctx context.Context, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return nil, err
	}
	if client.sourceName == "" {
		return nil, fmt.Errorf("rss source name is required")
	}
	if client.feedURL == "" {
		return nil, fmt.Errorf("rss feed url is required")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.feedURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "articleflow-parser/0.1")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET %s returned status %d", client.feedURL, response.StatusCode)
	}

	items, err := parseRSS(response.Body)
	if err != nil {
		return nil, err
	}
	terms := meaningfulTerms(query.Text)
	candidates := make([]parserv1.ArticleCandidate, 0, len(items))
	for _, item := range items {
		if !matchesTerms(item, terms) {
			continue
		}
		candidates = append(candidates, parserv1.ArticleCandidate{
			SourceName:  client.sourceName,
			ExternalID:  firstNonEmpty(item.GUID, item.Link),
			URL:         strings.TrimSpace(item.Link),
			Title:       strings.TrimSpace(item.Title),
			Summary:     strings.TrimSpace(item.Description),
			Content:     strings.TrimSpace(item.Description),
			Author:      strings.TrimSpace(item.Author),
			Tags:        cleanStrings(item.Categories),
			Language:    client.language,
			PublishedAt: parsePublishedAt(item.PubDate),
		})
		if query.Limit > 0 && len(candidates) >= query.Limit {
			break
		}
	}
	return candidates, nil
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        string   `xml:"guid"`
	Description string   `xml:"description"`
	Author      string   `xml:"author"`
	Categories  []string `xml:"category"`
	PubDate     string   `xml:"pubDate"`
}

func parseRSS(reader io.Reader) ([]rssItem, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var document rssDocument
	if err := xml.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	if len(document.Channel.Items) > 0 {
		return document.Channel.Items, nil
	}

	var atom atomDocument
	if err := xml.Unmarshal(payload, &atom); err != nil {
		return nil, err
	}
	items := make([]rssItem, 0, len(atom.Entries))
	for _, entry := range atom.Entries {
		items = append(items, rssItem{
			Title:       entry.Title,
			Link:        entry.LinkURL(),
			GUID:        entry.ID,
			Description: firstNonEmpty(entry.Summary, entry.Content),
			Author:      entry.Author.Name,
			Categories:  entry.Categories(),
			PubDate:     firstNonEmpty(entry.Published, entry.Updated),
		})
	}
	return items, nil
}

type atomDocument struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string         `xml:"title"`
	ID        string         `xml:"id"`
	Summary   string         `xml:"summary"`
	Content   string         `xml:"content"`
	Updated   string         `xml:"updated"`
	Published string         `xml:"published"`
	Author    atomAuthor     `xml:"author"`
	Links     []atomLink     `xml:"link"`
	Tags      []atomCategory `xml:"category"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

func (entry atomEntry) LinkURL() string {
	for _, link := range entry.Links {
		if strings.TrimSpace(link.Rel) == "" || link.Rel == "alternate" {
			return strings.TrimSpace(link.Href)
		}
	}
	if len(entry.Links) > 0 {
		return strings.TrimSpace(entry.Links[0].Href)
	}
	return ""
}

func (entry atomEntry) Categories() []string {
	categories := make([]string, 0, len(entry.Tags))
	for _, tag := range entry.Tags {
		if strings.TrimSpace(tag.Term) != "" {
			categories = append(categories, strings.TrimSpace(tag.Term))
		}
	}
	return categories
}

func matchesTerms(item rssItem, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	text := strings.ToLower(item.Title + " " + item.Description + " " + strings.Join(item.Categories, " "))
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func meaningfulTerms(text string) []string {
	seen := make(map[string]struct{})
	rawTerms := strings.Fields(strings.ToLower(text))
	terms := make([]string, 0, len(rawTerms))
	for _, term := range rawTerms {
		term = strings.Trim(term, " \t\n\r.,!?;:()[]{}\"'`«»")
		if len([]rune(term)) < 3 {
			continue
		}
		if _, ok := rssStopWords[term]; ok {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

var rssStopWords = map[string]struct{}{
	"and": {}, "for": {}, "the": {}, "with": {}, "или": {}, "как": {}, "что": {}, "для": {},
	"это": {}, "про": {}, "при": {}, "без": {}, "под": {}, "над": {}, "его": {}, "её": {},
	"она": {}, "они": {}, "оно": {}, "там": {}, "тут": {}, "все": {}, "всё": {},
}

func parsePublishedAt(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.RFC1123Z, time.RFC1123} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
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

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

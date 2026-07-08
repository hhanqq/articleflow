package vc

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
)

type structuredArticle struct {
	Title       string
	Summary     string
	Content     string
	URL         string
	Author      string
	Tags        []string
	PublishedAt time.Time
}

func ParseArticleHTML(reader io.Reader, articleURL string) (articlev1.Article, error) {
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return articlev1.Article{}, err
	}

	structured := parseStructuredArticle(document)
	stateArticle := parseInitialStateArticle(document)
	canonicalURL := firstNonEmpty(stateArticle.URL, structured.URL, attr(document, `link[rel="canonical"]`, "href"), articleURL)
	title := firstNonEmpty(stateArticle.Title, structured.Title, meta(document, `meta[property="og:title"]`), cleanText(document.Find("title").First().Text()))
	summary := firstNonEmpty(structured.Summary, meta(document, `meta[name="description"]`))
	content := firstNonEmpty(stateArticle.Content, structured.Content, cleanText(document.Find(".content .l-island-a, .content, article").First().Text()), summary)
	author := firstNonEmpty(stateArticle.Author, structured.Author, meta(document, `meta[name="author"]`))
	tags := firstNonEmptyStrings(stateArticle.Tags, structured.Tags)
	publishedAt := stateArticle.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = structured.PublishedAt
	}

	article := articlev1.Article{
		ID:          "vc:" + stableExternalID(canonicalURL),
		SourceName:  SourceName,
		ExternalID:  stableExternalID(canonicalURL),
		URL:         cleanURL(canonicalURL),
		Title:       cleanText(title),
		Summary:     cleanText(summary),
		Content:     cleanText(content),
		Author:      cleanText(author),
		Tags:        cleanStrings(tags),
		Language:    "ru",
		PublishedAt: publishedAt,
		ParsedAt:    time.Now().UTC(),
	}
	return article, article.Validate()
}

func parseInitialStateArticle(document *goquery.Document) structuredArticle {
	var article structuredArticle
	document.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		statePayload, ok := extractInitialState(script.Text())
		if !ok {
			return true
		}
		parsed, ok := parseInitialStatePayload(statePayload)
		if !ok {
			return true
		}
		article = parsed
		return false
	})
	return article
}

func extractInitialState(script string) (string, bool) {
	const marker = "window.__INITIAL_STATE__"
	index := strings.Index(script, marker)
	if index < 0 {
		return "", false
	}
	afterMarker := script[index+len(marker):]
	equalsIndex := strings.Index(afterMarker, "=")
	if equalsIndex < 0 {
		return "", false
	}
	afterEquals := strings.TrimSpace(afterMarker[equalsIndex+1:])
	objectStart := strings.Index(afterEquals, "{")
	if objectStart < 0 {
		return "", false
	}
	payload := afterEquals[objectStart:]
	end := jsonObjectEnd(payload)
	if end <= 0 {
		return "", false
	}
	return payload[:end], true
}

func jsonObjectEnd(value string) int {
	depth := 0
	inString := false
	escaped := false
	for index, char := range value {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index + 1
			}
		}
	}
	return -1
}

func parseInitialStatePayload(payload string) (structuredArticle, bool) {
	var state map[string]any
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return structuredArticle{}, false
	}
	for key, value := range state {
		if !strings.HasPrefix(key, "entry@") {
			continue
		}
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		return structuredArticle{
			Title:       asString(entry["title"]),
			Summary:     asString(entry["ogDescription"]),
			Content:     contentFromBlocks(entry["blocks"]),
			URL:         asString(entry["url"]),
			Author:      authorName(entry["author"]),
			Tags:        keywords(entry["keywords"]),
			PublishedAt: unixTime(entry["date"]),
		}, true
	}
	return structuredArticle{}, false
}

func contentFromBlocks(value any) string {
	blocks, ok := value.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, blockValue := range blocks {
		block, ok := blockValue.(map[string]any)
		if !ok {
			continue
		}
		data, _ := block["data"].(map[string]any)
		switch asString(block["type"]) {
		case "text":
			parts = append(parts, htmlFragmentText(asString(data["text"])))
		case "list":
			items, _ := data["items"].([]any)
			for _, item := range items {
				parts = append(parts, htmlFragmentText(asString(item)))
			}
		}
	}
	return cleanText(strings.Join(parts, " "))
}

func htmlFragmentText(fragment string) string {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return ""
	}
	document, err := goquery.NewDocumentFromReader(strings.NewReader("<body>" + fragment + "</body>"))
	if err != nil {
		return cleanText(fragment)
	}
	return cleanText(document.Find("body").Text())
}

func parseStructuredArticle(document *goquery.Document) structuredArticle {
	var article structuredArticle
	document.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		parsed, ok := parseJSONLDArticle(script.Text())
		if !ok {
			return true
		}
		article = parsed
		return false
	})
	return article
}

func parseJSONLDArticle(payload string) (structuredArticle, bool) {
	var document any
	if err := json.Unmarshal([]byte(payload), &document); err != nil {
		return structuredArticle{}, false
	}
	node, ok := findArticleNode(document)
	if !ok {
		return structuredArticle{}, false
	}
	return structuredArticle{
		Title:       stringField(node, "headline", "name"),
		Summary:     stringField(node, "description"),
		Content:     stringField(node, "text", "articleBody"),
		URL:         stringField(node, "url"),
		Author:      authorName(node["author"]),
		Tags:        keywords(node["keywords"]),
		PublishedAt: parsePublishedAt(stringField(node, "datePublished")),
	}, true
}

func findArticleNode(value any) (map[string]any, bool) {
	node, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	if graph, ok := node["@graph"].([]any); ok {
		for _, item := range graph {
			itemNode, ok := item.(map[string]any)
			if ok && isArticleNode(itemNode) {
				return itemNode, true
			}
		}
	}
	if isArticleNode(node) {
		return node, true
	}
	return nil, false
}

func isArticleNode(node map[string]any) bool {
	value := node["@type"]
	switch typed := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(typed), "article")
	case []any:
		for _, item := range typed {
			if strings.Contains(strings.ToLower(asString(item)), "article") {
				return true
			}
		}
	}
	return false
}

func stringField(node map[string]any, keys ...string) string {
	for _, key := range keys {
		value := asString(node[key])
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func authorName(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		return asString(typed["name"])
	case []any:
		for _, item := range typed {
			name := authorName(item)
			if strings.TrimSpace(name) != "" {
				return name
			}
		}
	}
	return ""
}

func keywords(value any) []string {
	switch typed := value.(type) {
	case string:
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == '#'
		})
		return cleanStrings(parts)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, asString(item))
		}
		return cleanStrings(values)
	}
	return nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func parsePublishedAt(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func unixTime(value any) time.Time {
	switch typed := value.(type) {
	case float64:
		if typed <= 0 {
			return time.Time{}
		}
		return time.Unix(int64(typed), 0).UTC()
	case int64:
		if typed <= 0 {
			return time.Time{}
		}
		return time.Unix(typed, 0).UTC()
	}
	return time.Time{}
}

func meta(document *goquery.Document, selector string) string {
	return attr(document, selector, "content")
}

func attr(document *goquery.Document, selector string, name string) string {
	value, _ := document.Find(selector).First().Attr(name)
	return strings.TrimSpace(value)
}

func cleanURL(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func stableExternalID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func firstNonEmptyStrings(values ...[]string) []string {
	for _, value := range values {
		if len(cleanStrings(value)) > 0 {
			return value
		}
	}
	return nil
}

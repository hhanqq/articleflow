package dzen

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"strconv"
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

var (
	dzenEpochTimePatterns = []*regexp.Regexp{
		regexp.MustCompile(`"publishTime"\s*:\s*([0-9]{10,13})`),
		regexp.MustCompile(`"addTime"\s*:\s*([0-9]{10,13})`),
	}
	dzenPublishDatePattern = regexp.MustCompile(`"publishDate"\s*:\s*"([^"]+)"`)
)

func ParseArticleHTML(reader io.Reader, articleURL string) (articlev1.Article, error) {
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return articlev1.Article{}, err
	}
	structured := parseStructuredArticle(document)
	canonicalURL := cleanURL(firstNonEmpty(structured.URL, attr(document, `link[rel="canonical"]`, "href"), articleURL))
	title := cleanText(firstNonEmpty(structured.Title, meta(document, `meta[property="og:title"]`), cleanText(document.Find("title").First().Text())))
	summary := cleanText(firstNonEmpty(structured.Summary, meta(document, `meta[name="description"]`), meta(document, `meta[property="og:description"]`)))
	content := cleanText(firstNonEmpty(structured.Content, cleanText(document.Find("article").First().Text()), summary))
	author := cleanText(firstNonEmpty(structured.Author, meta(document, `meta[name="author"]`)))
	article := articlev1.Article{
		ID:          SourceName + ":" + stableExternalID(canonicalURL),
		SourceName:  SourceName,
		ExternalID:  stableExternalID(canonicalURL),
		URL:         canonicalURL,
		Title:       title,
		Summary:     summary,
		Content:     content,
		Author:      author,
		Tags:        cleanStrings(structured.Tags),
		Language:    "ru",
		PublishedAt: structured.PublishedAt,
		ParsedAt:    time.Now().UTC(),
	}
	return article, article.Validate()
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
	if article.PublishedAt.IsZero() {
		article.PublishedAt = parseEmbeddedDzenPublishedAt(document)
	}
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
		Title:       firstNonEmpty(asString(node["headline"]), asString(node["name"])),
		Summary:     asString(node["description"]),
		Content:     asString(node["articleBody"]),
		URL:         asString(node["url"]),
		Author:      authorName(node["author"]),
		Tags:        keywords(node["keywords"]),
		PublishedAt: parseTime(asString(node["datePublished"])),
	}, true
}

func findArticleNode(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if isArticleType(typed["@type"]) {
			return typed, true
		}
		if graph, ok := typed["@graph"]; ok {
			return findArticleNode(graph)
		}
		for _, child := range typed {
			if node, ok := findArticleNode(child); ok {
				return node, true
			}
		}
	case []any:
		for _, child := range typed {
			if node, ok := findArticleNode(child); ok {
				return node, true
			}
		}
	}
	return nil, false
}

func isArticleType(value any) bool {
	switch typed := value.(type) {
	case string:
		return typed == "Article" || typed == "NewsArticle" || typed == "BlogPosting"
	case []any:
		for _, item := range typed {
			if isArticleType(item) {
				return true
			}
		}
	}
	return false
}

func authorName(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		return asString(typed["name"])
	case []any:
		names := make([]string, 0, len(typed))
		for _, item := range typed {
			if name := authorName(item); name != "" {
				names = append(names, name)
			}
		}
		return strings.Join(names, ", ")
	}
	return ""
}

func keywords(value any) []string {
	switch typed := value.(type) {
	case string:
		return strings.Split(typed, ",")
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if keyword := asString(item); keyword != "" {
				result = append(result, keyword)
			}
		}
		return result
	}
	return nil
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseEmbeddedDzenPublishedAt(document *goquery.Document) time.Time {
	var publishedAt time.Time
	document.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		text := script.Text()
		for _, pattern := range dzenEpochTimePatterns {
			if parsed := parseUnixTimestamp(pattern.FindStringSubmatch(text)); !parsed.IsZero() {
				publishedAt = parsed
				return false
			}
		}
		if match := dzenPublishDatePattern.FindStringSubmatch(text); len(match) == 2 {
			if parsed := parseTime(match[1]); !parsed.IsZero() {
				publishedAt = parsed
				return false
			}
		}
		return true
	})
	return publishedAt
}

func parseUnixTimestamp(match []string) time.Time {
	if len(match) != 2 {
		return time.Time{}
	}
	value, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || value <= 0 {
		return time.Time{}
	}
	if value > 9999999999 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func attr(document *goquery.Document, selector string, name string) string {
	value, _ := document.Find(selector).First().Attr(name)
	return strings.TrimSpace(value)
}

func meta(document *goquery.Document, selector string) string {
	return attr(document, selector, "content")
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	}
	return ""
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = cleanText(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func cleanURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func stableExternalID(value string) string {
	parsed, err := url.Parse(value)
	if err == nil && strings.Trim(parsed.Path, "/") != "" {
		return strings.Trim(parsed.Path, "/")
	}
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

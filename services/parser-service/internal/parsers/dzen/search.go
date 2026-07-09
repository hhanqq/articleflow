package dzen

import (
	"bytes"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var absoluteDzenArticleURLPattern = regexp.MustCompile(`https://dzen\.ru/(?:a|media)/[^"'<>\s\\]+`)

func ParseSearchHTML(reader io.Reader, searchURL string, publicBaseURL string, limit int) ([]string, error) {
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	base := firstNonEmpty(publicBaseURL, searchURL, "https://dzen.ru")
	capacity := limit
	if capacity <= 0 {
		capacity = 16
	}
	urls := make([]string, 0, capacity)
	seen := make(map[string]bool)
	addURL := func(rawURL string) bool {
		cleaned := normalizeDzenArticleURL(rawURL, base)
		if cleaned == "" || seen[cleaned] {
			return true
		}
		seen[cleaned] = true
		urls = append(urls, cleaned)
		return limit <= 0 || len(urls) < limit
	}
	document.Find("a[href]").EachWithBreak(func(_ int, link *goquery.Selection) bool {
		href, _ := link.Attr("href")
		return addURL(href)
	})
	if limit > 0 && len(urls) >= limit {
		return urls, nil
	}
	for _, embeddedURL := range embeddedDzenArticleURLs(string(payload)) {
		if !addURL(embeddedURL) {
			break
		}
	}
	return urls, nil
}

func embeddedDzenArticleURLs(payload string) []string {
	normalized := strings.NewReplacer(`\/`, `/`, `\u002F`, `/`, `\u002f`, `/`).Replace(payload)
	return absoluteDzenArticleURLPattern.FindAllString(normalized, -1)
}

func normalizeDzenArticleURL(rawURL string, baseURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(rawURL, "javascript:") {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	if !isDzenArticlePath(resolved.Path) {
		return ""
	}
	resolved.RawQuery = ""
	resolved.Fragment = ""
	if resolved.Scheme == "" {
		resolved.Scheme = "https"
	}
	return resolved.String()
}

func isDzenArticlePath(path string) bool {
	path = strings.Trim(strings.ToLower(path), "/")
	if strings.HasPrefix(path, "media/zen/login") {
		return false
	}
	return strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "media/")
}

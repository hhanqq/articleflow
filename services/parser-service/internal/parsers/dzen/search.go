package dzen

import (
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseSearchHTML(reader io.Reader, searchURL string, publicBaseURL string, limit int) ([]string, error) {
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}
	base := firstNonEmpty(publicBaseURL, searchURL, "https://dzen.ru")
	urls := make([]string, 0, limit)
	seen := make(map[string]bool)
	document.Find("a[href]").EachWithBreak(func(_ int, link *goquery.Selection) bool {
		href, _ := link.Attr("href")
		cleaned := normalizeDzenArticleURL(href, base)
		if cleaned == "" || seen[cleaned] {
			return true
		}
		seen[cleaned] = true
		urls = append(urls, cleaned)
		return limit <= 0 || len(urls) < limit
	})
	return urls, nil
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

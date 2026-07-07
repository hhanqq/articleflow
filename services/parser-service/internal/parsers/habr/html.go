package habr

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/url"
	"strings"
	"time"

	articlev1 "github.com/hanq/articleflow/contracts/article/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	"github.com/PuerkitoBio/goquery"
)

const baseURL = "https://habr.com"

func ParseSearchHTML(reader io.Reader, query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	query = query.Normalize()
	candidates := make([]parserv1.ArticleCandidate, 0)
	document.Find("article").Each(func(_ int, article *goquery.Selection) {
		titleLink := article.Find(".tm-title__link").First()
		title := cleanText(titleLink.Text())
		href, _ := titleLink.Attr("href")
		articleURL := absoluteURL(href)
		if title == "" || articleURL == "" {
			return
		}

		candidate := parserv1.ArticleCandidate{
			SourceName:  SourceName,
			ExternalID:  stableExternalID(articleURL),
			URL:         articleURL,
			Title:       title,
			Summary:     cleanText(article.Find(".article-formatted-body").First().Text()),
			Author:      cleanText(article.Find(".tm-user-info__username").First().Text()),
			Tags:        collectTags(article),
			Language:    query.Language,
			PublishedAt: parseDatetimeAttr(article.Find("time").First()),
		}
		candidates = append(candidates, candidate)
	})

	if query.Limit > 0 && len(candidates) > query.Limit {
		candidates = candidates[:query.Limit]
	}
	return candidates, nil
}

func ParseArticleHTML(reader io.Reader, articleURL string) (articlev1.Article, error) {
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return articlev1.Article{}, err
	}

	title := cleanText(document.Find("h1").First().Text())
	if title == "" {
		title, _ = document.Find(`meta[property="og:title"]`).Attr("content")
		title = cleanText(title)
	}

	article := articlev1.Article{
		ID:          "article-" + stableExternalID(articleURL),
		SourceName:  SourceName,
		ExternalID:  stableExternalID(articleURL),
		URL:         articleURL,
		Title:       title,
		Content:     cleanText(document.Find("#post-content-body").First().Text()),
		Author:      cleanText(document.Find(".tm-user-info__username").First().Text()),
		Tags:        collectTags(document.Selection),
		Language:    "ru",
		PublishedAt: parseDatetimeAttr(document.Find("time").First()),
		ParsedAt:    time.Now().UTC(),
	}
	return article, article.Validate()
}

func absoluteURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	base, _ := url.Parse(baseURL)
	return base.ResolveReference(parsed).String()
}

func collectTags(selection *goquery.Selection) []string {
	tags := make([]string, 0)
	selection.Find(".tm-publication-hub__link").Each(func(_ int, tag *goquery.Selection) {
		value := cleanText(tag.Text())
		if value != "" {
			tags = append(tags, value)
		}
	})
	return tags
}

func parseDatetimeAttr(selection *goquery.Selection) time.Time {
	value, _ := selection.Attr("datetime")
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func stableExternalID(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}


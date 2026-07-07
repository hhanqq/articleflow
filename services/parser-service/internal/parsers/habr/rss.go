package habr

import (
	"encoding/xml"
	"io"
	"strings"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
)

const SourceName = "habr"

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

func ParseRSS(reader io.Reader) ([]eventsv1.ArticleDiscoveredEvent, error) {
	var document rssDocument
	if err := xml.NewDecoder(reader).Decode(&document); err != nil {
		return nil, err
	}

	events := make([]eventsv1.ArticleDiscoveredEvent, 0, len(document.Channel.Items))
	now := time.Now().UTC()
	for _, item := range document.Channel.Items {
		publishedAt, _ := time.Parse(time.RFC1123Z, strings.TrimSpace(item.PubDate))
		event := eventsv1.ArticleDiscoveredEvent{
			SourceName:   SourceName,
			ExternalID:   firstNonEmpty(item.GUID, item.Link),
			URL:          strings.TrimSpace(item.Link),
			Title:        strings.TrimSpace(item.Title),
			Summary:      strings.TrimSpace(item.Description),
			Author:       strings.TrimSpace(item.Author),
			Tags:         cleanStrings(item.Categories),
			Language:     "ru",
			PublishedAt:  publishedAt,
			DiscoveredAt: now,
		}
		events = append(events, event)
	}
	return events, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
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


package usecase

import (
	"sort"
	"strings"
	"time"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type Ranker struct {
	now func() time.Time
}

func NewRanker() *Ranker {
	return &Ranker{now: time.Now}
}

func (ranker *Ranker) Rank(query parserv1.SearchQuery, items []feedv1.FeedItem) []feedv1.FeedItem {
	terms := strings.Fields(strings.ToLower(query.Text))
	ranked := append([]feedv1.FeedItem(nil), items...)
	for index := range ranked {
		ranked[index].Score = ranker.score(terms, ranked[index])
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

func (ranker *Ranker) score(terms []string, item feedv1.FeedItem) float64 {
	score := 0.0
	text := strings.ToLower(item.Title + " " + item.Summary + " " + strings.Join(item.Tags, " "))
	for _, term := range terms {
		if strings.Contains(text, term) {
			score += 10
		}
	}
	if item.SourceName == "habr" {
		score += 3
	}
	if !item.PublishedAt.IsZero() {
		ageHours := ranker.now().UTC().Sub(item.PublishedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
		score += 20 / (1 + ageHours/24)
	}
	return score
}


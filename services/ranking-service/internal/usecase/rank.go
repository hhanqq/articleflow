package usecase

import (
	"fmt"
	"sort"
	"strings"
	"time"

	eventsv1 "github.com/hanq/articleflow/contracts/events/v1"
	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type Ranker struct {
	now     func() time.Time
	signals *SignalStore
}

func NewRanker() *Ranker {
	return &Ranker{now: time.Now}
}

func NewRankerWithSignals(signals *SignalStore) *Ranker {
	return &Ranker{now: time.Now, signals: signals}
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

func (ranker *Ranker) RankDiscovered(event eventsv1.ArticleDiscoveredEvent) eventsv1.FeedItemScoredEvent {
	item := feedv1.FeedItem{
		ArticleID:   articleID(event),
		Title:       event.Title,
		Summary:     event.Summary,
		SourceName:  event.SourceName,
		URL:         event.URL,
		Tags:        event.Tags,
		PublishedAt: event.PublishedAt,
	}
	score := ranker.score(nil, item)
	if ranker.signals != nil {
		ranker.signals.RememberArticle(item)
	}
	return eventsv1.FeedItemScoredEvent{
		ArticleID:   item.ArticleID,
		SourceName:  item.SourceName,
		URL:         item.URL,
		Title:       item.Title,
		Summary:     item.Summary,
		Tags:        append([]string(nil), item.Tags...),
		Score:       score,
		PublishedAt: item.PublishedAt,
		ScoredAt:    ranker.now().UTC(),
	}
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
	if ranker.signals != nil {
		score = ranker.signals.ApplyToScore(score, item)
	}
	return score
}

func articleID(event eventsv1.ArticleDiscoveredEvent) string {
	if event.SourceName != "" && event.ExternalID != "" {
		return fmt.Sprintf("%s:%s", event.SourceName, event.ExternalID)
	}
	return event.URL
}

type ReactionScorer struct{}

func NewReactionScorer() ReactionScorer {
	return ReactionScorer{}
}

func (ReactionScorer) Apply(baseScore float64, reactions []userv1.UserReaction) float64 {
	score := baseScore
	for _, reaction := range reactions {
		switch reaction.Type {
		case userv1.ReactionLike:
			score += 4
		case userv1.ReactionSave:
			score += 8
		case userv1.ReactionOpen:
			score += 2
		case userv1.ReactionSkip:
			score -= 3
		case userv1.ReactionDislike:
			score -= 8
		}
	}
	if score < 0 {
		return 0
	}
	return score
}

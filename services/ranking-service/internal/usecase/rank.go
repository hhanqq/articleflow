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
		score, reasons := ranker.scoreWithReasons(terms, ranked[index])
		ranked[index].Score = score
		ranked[index].ScoreReasons = reasons
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
	score, reasons := ranker.scoreWithReasons(nil, item)
	if ranker.signals != nil {
		ranker.signals.RememberArticle(item)
	}
	return eventsv1.FeedItemScoredEvent{
		ArticleID:    item.ArticleID,
		SourceName:   item.SourceName,
		URL:          item.URL,
		Title:        item.Title,
		Summary:      item.Summary,
		Tags:         append([]string(nil), item.Tags...),
		Score:        score,
		ScoreReasons: reasons,
		PublishedAt:  item.PublishedAt,
		ScoredAt:     ranker.now().UTC(),
	}
}

func (ranker *Ranker) score(terms []string, item feedv1.FeedItem) float64 {
	score, _ := ranker.scoreWithReasons(terms, item)
	return score
}

func (ranker *Ranker) scoreWithReasons(terms []string, item feedv1.FeedItem) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 4)
	text := strings.ToLower(item.Title + " " + item.Summary + " " + strings.Join(item.Tags, " "))
	for _, term := range terms {
		if strings.Contains(text, term) {
			score += 10
			reasons = append(reasons, "query_match:"+term)
		}
	}
	if item.SourceName == "habr" {
		score += 3
		reasons = append(reasons, "source_boost:habr")
	}
	if !item.PublishedAt.IsZero() {
		ageHours := ranker.now().UTC().Sub(item.PublishedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
		freshnessScore := 20 / (1 + ageHours/24)
		score += freshnessScore
		reasons = append(reasons, "freshness")
	}
	if ranker.signals != nil {
		beforeSignals := score
		score = ranker.signals.ApplyToScore(score, item)
		if score != beforeSignals {
			reasons = append(reasons, "reaction_signals")
		}
	}
	return score, compactReasons(reasons)
}

func compactReasons(reasons []string) []string {
	seen := make(map[string]bool, len(reasons))
	compacted := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" || seen[reason] {
			continue
		}
		seen[reason] = true
		compacted = append(compacted, reason)
	}
	return compacted
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

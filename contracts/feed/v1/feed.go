package feedv1

import "time"

type FeedItem struct {
	ArticleID    string
	Title        string
	Summary      string
	SourceName   string
	URL          string
	Tags         []string
	Score        float64
	ScoreReasons []string
	Reaction     string
	Saved        bool
	PublishedAt  time.Time
}

type GetFeedRequest struct {
	UserID string
	Limit  int
	Cursor string
}

type GetFeedResponse struct {
	Items      []FeedItem
	NextCursor string
}

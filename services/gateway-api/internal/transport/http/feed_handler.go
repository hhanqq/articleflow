package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
)

type FeedProvider interface {
	List(limit int) []feedv1.FeedItem
}

type FeedSearchProvider interface {
	Search(query parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error)
}

type FeedParserJobClient interface {
	StartAsync(ctx context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error)
}

type UserReactionProvider interface {
	ListUserReactions(ctx context.Context, userID string) ([]userv1.UserReaction, error)
}

type FeedHandlerDependencies struct {
	FeedProvider         FeedProvider
	SearchProvider       FeedSearchProvider
	ParserJobClient      FeedParserJobClient
	UserReactionProvider UserReactionProvider
	RefillMinItems       int
	RefillLimit          int
	CacheTTL             time.Duration
}

type FeedResponse struct {
	Items         []feedv1.FeedItem   `json:"items"`
	Mode          string              `json:"mode,omitempty"`
	ReturnedCount int                 `json:"returned_count"`
	NextCursor    string              `json:"next_cursor,omitempty"`
	RefillStarted bool                `json:"refill_started,omitempty"`
	RefillJob     *parserv1.ParserJob `json:"refill_job,omitempty"`
}

func NewFeedHandler(provider FeedProvider) http.Handler {
	return NewFeedHandlerWithRefill(FeedHandlerDependencies{FeedProvider: provider})
}

func NewFeedHandlerWithRefill(dependencies FeedHandlerDependencies) http.Handler {
	cache := newFeedResponseCache(dependencies.CacheTTL)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseLimit(request)
		queryText := strings.TrimSpace(request.URL.Query().Get("query"))
		if queryText != "" && dependencies.SearchProvider != nil {
			handleQueryFeed(response, request, dependencies, cache, queryText, limit)
			return
		}
		items := personalizedFeedItems(request, dependencies, dependencies.FeedProvider.List(limit))
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(FeedResponse{
			Items:         items,
			Mode:          "default",
			ReturnedCount: len(items),
		})
	})
}

func handleQueryFeed(response http.ResponseWriter, request *http.Request, dependencies FeedHandlerDependencies, cache *feedResponseCache, queryText string, limit int) {
	cacheKey := request.URL.RawQuery
	if cache != nil && !wantsRefill(request) {
		if payload, ok := cache.get(cacheKey); ok {
			writeFeedResponse(response, payload)
			return
		}
	}
	query := parserv1.SearchQuery{
		Text:    queryText,
		Sources: parseCSVQueryValues(request, "sources"),
		Limit:   limit,
		Offset:  parseFeedCursor(request.URL.Query().Get("cursor")),
	}
	fromDate, ok := parseOptionalTimeQuery(response, request, "from_date")
	if !ok {
		return
	}
	toDate, ok := parseOptionalTimeQuery(response, request, "to_date")
	if !ok {
		return
	}
	query.FromDate = fromDate
	query.ToDate = toDate
	query = query.Normalize()
	candidates, err := dependencies.SearchProvider.Search(query)
	if err != nil {
		http.Error(response, "feed search failed", http.StatusBadGateway)
		return
	}
	items := personalizedFeedItems(request, dependencies, feedItemsFromCandidates(candidates))
	payload := FeedResponse{
		Items:         items,
		Mode:          "query",
		ReturnedCount: len(items),
		NextCursor:    nextFeedCursor(query.Offset, len(items), limit),
	}
	if shouldStartRefill(request, dependencies, len(items), limit) {
		refillQuery := query
		refillQuery.Limit = refillLimit(dependencies.RefillLimit, limit)
		job, err := dependencies.ParserJobClient.StartAsync(request.Context(), refillQuery.Normalize())
		if err == nil {
			payload.RefillStarted = true
			payload.RefillJob = &job
		}
	}
	if cache != nil && !payload.RefillStarted {
		cache.set(cacheKey, payload)
	}
	writeFeedResponse(response, payload)
}

func shouldStartRefill(request *http.Request, dependencies FeedHandlerDependencies, itemCount int, limit int) bool {
	if dependencies.ParserJobClient == nil {
		return false
	}
	if !wantsRefill(request) {
		return false
	}
	minItems := dependencies.RefillMinItems
	if minItems <= 0 {
		minItems = limit / 2
	}
	if minItems <= 0 {
		minItems = 10
	}
	return itemCount < minItems
}

func wantsRefill(request *http.Request) bool {
	rawRefill := strings.ToLower(strings.TrimSpace(request.URL.Query().Get("refill")))
	return rawRefill == "true" || rawRefill == "1" || rawRefill == "yes"
}

func refillLimit(configured int, feedLimit int) int {
	if configured > 0 {
		if configured > 100 {
			return 100
		}
		return configured
	}
	limit := feedLimit * 5
	if limit < 80 {
		limit = 80
	}
	if limit > 100 {
		limit = 100
	}
	return limit
}

func feedItemsFromCandidates(candidates []parserv1.ArticleCandidate) []feedv1.FeedItem {
	items := make([]feedv1.FeedItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, feedv1.FeedItem{
			ArticleID:   feedArticleID(candidate),
			Title:       candidate.Title,
			Summary:     candidate.Summary,
			SourceName:  candidate.SourceName,
			URL:         candidate.URL,
			Tags:        append([]string(nil), candidate.Tags...),
			PublishedAt: candidate.PublishedAt,
		})
	}
	return items
}

func feedArticleID(candidate parserv1.ArticleCandidate) string {
	if candidate.SourceName != "" && candidate.ExternalID != "" {
		return candidate.SourceName + ":" + candidate.ExternalID
	}
	return candidate.URL
}

func parseCSVQueryValues(request *http.Request, key string) []string {
	rawValues := request.URL.Query()[key]
	values := make([]string, 0, len(rawValues))
	seen := make(map[string]bool)
	for _, rawValue := range rawValues {
		for _, part := range strings.Split(rawValue, ",") {
			value := strings.TrimSpace(part)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

func parseFeedCursor(rawCursor string) int {
	rawCursor = strings.TrimSpace(rawCursor)
	if rawCursor == "" {
		return 0
	}
	rawOffset, ok := strings.CutPrefix(rawCursor, "offset:")
	if !ok {
		return 0
	}
	offset, err := strconv.Atoi(strings.TrimSpace(rawOffset))
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func parseOptionalTimeQuery(response http.ResponseWriter, request *http.Request, key string) (*time.Time, bool) {
	rawValue := strings.TrimSpace(request.URL.Query().Get(key))
	if rawValue == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, rawValue)
	if err != nil {
		http.Error(response, key+" must be an RFC3339 timestamp", http.StatusBadRequest)
		return nil, false
	}
	parsed = parsed.UTC()
	return &parsed, true
}

func nextFeedCursor(currentOffset int, returnedCount int, limit int) string {
	if returnedCount <= 0 || limit <= 0 || returnedCount < limit {
		return ""
	}
	return "offset:" + strconv.Itoa(currentOffset+returnedCount)
}

func personalizedFeedItems(request *http.Request, dependencies FeedHandlerDependencies, items []feedv1.FeedItem) []feedv1.FeedItem {
	userID := UserIDFromRequest(request)
	if userID == "" || dependencies.UserReactionProvider == nil || len(items) == 0 {
		return items
	}
	reactions, err := dependencies.UserReactionProvider.ListUserReactions(request.Context(), userID)
	if err != nil || len(reactions) == 0 {
		return items
	}
	return applyUserReactions(items, reactions)
}

func applyUserReactions(items []feedv1.FeedItem, reactions []userv1.UserReaction) []feedv1.FeedItem {
	byArticle := latestReactionsByArticle(reactions)
	result := make([]feedv1.FeedItem, 0, len(items))
	for _, item := range items {
		reaction, ok := byArticle[item.ArticleID]
		if ok && hidesFeedItem(reaction.Type) {
			continue
		}
		if ok {
			item.Reaction = string(reaction.Type)
			item.Saved = reaction.Type == userv1.ReactionSave
		}
		result = append(result, item)
	}
	return result
}

func latestReactionsByArticle(reactions []userv1.UserReaction) map[string]userv1.UserReaction {
	byArticle := make(map[string]userv1.UserReaction)
	for _, reaction := range reactions {
		if strings.TrimSpace(reaction.ArticleID) == "" {
			continue
		}
		current, ok := byArticle[reaction.ArticleID]
		currentPriority := reactionPriority(current.Type)
		nextPriority := reactionPriority(reaction.Type)
		if !ok || nextPriority > currentPriority || nextPriority == currentPriority && reaction.CreatedAt.After(current.CreatedAt) {
			byArticle[reaction.ArticleID] = reaction
		}
	}
	return byArticle
}

func hidesFeedItem(reactionType userv1.ReactionType) bool {
	return reactionType == userv1.ReactionSkip || reactionType == userv1.ReactionDislike
}

func reactionPriority(reactionType userv1.ReactionType) int {
	switch reactionType {
	case userv1.ReactionSkip, userv1.ReactionDislike:
		return 3
	case userv1.ReactionSave:
		return 2
	case userv1.ReactionLike:
		return 1
	default:
		return 0
	}
}

func parseLimit(request *http.Request) int {
	limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func writeFeedResponse(response http.ResponseWriter, payload FeedResponse) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(payload)
}

type cachedFeedResponse struct {
	payload   FeedResponse
	expiresAt time.Time
}

type feedResponseCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cachedFeedResponse
	now     func() time.Time
}

func newFeedResponseCache(ttl time.Duration) *feedResponseCache {
	if ttl <= 0 {
		return nil
	}
	return &feedResponseCache{
		ttl:     ttl,
		entries: make(map[string]cachedFeedResponse),
		now:     time.Now,
	}
}

func (cache *feedResponseCache) get(key string) (FeedResponse, bool) {
	cache.mu.RLock()
	item, ok := cache.entries[key]
	cache.mu.RUnlock()
	if !ok || cache.now().After(item.expiresAt) {
		if ok {
			cache.mu.Lock()
			delete(cache.entries, key)
			cache.mu.Unlock()
		}
		return FeedResponse{}, false
	}
	return item.payload, true
}

func (cache *feedResponseCache) set(key string, payload FeedResponse) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.entries[key] = cachedFeedResponse{payload: payload, expiresAt: cache.now().Add(cache.ttl)}
}

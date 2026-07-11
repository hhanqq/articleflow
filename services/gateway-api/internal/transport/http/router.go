package httptransport

import (
	"net/http"
	"time"

	"github.com/hanq/articleflow/packages/observability"
)

type RouterDependencies struct {
	ServiceName          string
	ArticleProvider      ArticleProvider
	FeedProvider         FeedProvider
	SearchProvider       SearchProvider
	ParserJobClient      ParserJobClient
	ParserSourceClient   ParserSourceClient
	UserReactionProvider UserReactionProvider
	UserProfileProvider  UserProfileProvider
	ReactionRecorder     ReactionRecorder
	RateLimitPerMinute   int
	FeedCacheTTL         time.Duration
}

func NewRouter(dependencies RouterDependencies) http.Handler {
	mux := http.NewServeMux()
	metrics := observability.NewMetricsRegistry()
	metrics.Inc("articleflow_gateway_info")
	mux.Handle("/healthz", NewHealthHandler(dependencies.ServiceName))
	mux.Handle("/metrics", observability.NewPrometheusHandler(dependencies.ServiceName, metrics))
	mux.Handle("/api/v1/me", NewMeHandler(dependencies.UserProfileProvider))
	mux.Handle("/api/v1/feed", NewFeedHandlerWithRefill(FeedHandlerDependencies{
		FeedProvider:         dependencies.FeedProvider,
		SearchProvider:       dependencies.SearchProvider,
		ParserJobClient:      dependencies.ParserJobClient,
		UserReactionProvider: dependencies.UserReactionProvider,
		CacheTTL:             dependencies.FeedCacheTTL,
	}))
	mux.Handle("/api/v1/articles", NewArticleHandler(dependencies.ArticleProvider))
	mux.Handle("/api/v1/search", NewSearchHandler(dependencies.SearchProvider))
	mux.Handle("/api/v1/search/sources", NewSearchSourcesHandler(dependencies.ParserSourceClient))
	mux.Handle("/api/v1/search/sources/", NewSearchSourcesHandler(dependencies.ParserSourceClient))
	mux.Handle("/api/v1/search/jobs", NewSearchJobsHandler(dependencies.ParserJobClient))
	mux.Handle("/api/v1/search/jobs/", NewSearchJobsHandler(dependencies.ParserJobClient))
	mux.Handle("/api/v1/reactions", NewReactionHandler(dependencies.ReactionRecorder))
	handler := observability.InstrumentHTTPRequests(metrics, mux)
	handler = NewSessionMiddleware(handler)
	handler = NewRateLimitMiddleware(dependencies.RateLimitPerMinute, time.Minute, handler)
	return withCORS(handler)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Vary", "Origin")
			response.Header().Set("Access-Control-Allow-Credentials", "true")
			response.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
			response.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Articleflow-User-ID")
		}
		if request.Method == http.MethodOptions {
			response.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(response, request)
	})
}

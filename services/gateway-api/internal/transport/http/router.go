package httptransport

import (
	"net/http"

	"github.com/hanq/articleflow/packages/observability"
)

type RouterDependencies struct {
	ServiceName        string
	ArticleProvider    ArticleProvider
	FeedProvider       FeedProvider
	SearchProvider     SearchProvider
	ParserJobClient    ParserJobClient
	ParserSourceClient ParserSourceClient
	ReactionRecorder   ReactionRecorder
}

func NewRouter(dependencies RouterDependencies) http.Handler {
	mux := http.NewServeMux()
	metrics := observability.NewMetricsRegistry()
	metrics.Inc("articleflow_gateway_info")
	mux.Handle("/healthz", NewHealthHandler(dependencies.ServiceName))
	mux.Handle("/metrics", observability.NewPrometheusHandler(dependencies.ServiceName, metrics))
	mux.Handle("/api/v1/feed", NewFeedHandler(dependencies.FeedProvider))
	mux.Handle("/api/v1/articles", NewArticleHandler(dependencies.ArticleProvider))
	mux.Handle("/api/v1/search", NewSearchHandler(dependencies.SearchProvider))
	mux.Handle("/api/v1/search/sources", NewSearchSourcesHandler(dependencies.ParserSourceClient))
	mux.Handle("/api/v1/search/jobs", NewSearchJobsHandler(dependencies.ParserJobClient))
	mux.Handle("/api/v1/search/jobs/", NewSearchJobsHandler(dependencies.ParserJobClient))
	mux.Handle("/api/v1/reactions", NewReactionHandler(dependencies.ReactionRecorder))
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Vary", "Origin")
			response.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			response.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		}
		if request.Method == http.MethodOptions {
			response.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(response, request)
	})
}

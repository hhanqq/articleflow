package httptransport

import "net/http"

type RouterDependencies struct {
	ServiceName      string
	FeedProvider     FeedProvider
	SearchProvider   SearchProvider
	ReactionRecorder ReactionRecorder
}

func NewRouter(dependencies RouterDependencies) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", NewHealthHandler(dependencies.ServiceName))
	mux.Handle("/api/v1/feed", NewFeedHandler(dependencies.FeedProvider))
	mux.Handle("/api/v1/search", NewSearchHandler(dependencies.SearchProvider))
	mux.Handle("/api/v1/reactions", NewReactionHandler(dependencies.ReactionRecorder))
	return mux
}

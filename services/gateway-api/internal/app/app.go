package app

import (
	"context"
	"errors"
	"net/http"

	feedv1 "github.com/hanq/articleflow/contracts/feed/v1"
	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	"github.com/hanq/articleflow/services/gateway-api/internal/config"
	httptransport "github.com/hanq/articleflow/services/gateway-api/internal/transport/http"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Handler() http.Handler {
	return httptransport.NewRouter(httptransport.RouterDependencies{
		ServiceName:      app.cfg.ServiceName,
		FeedProvider:     emptyFeedProvider{},
		SearchProvider:   emptySearchProvider{},
		ReactionRecorder: noopReactionRecorder{},
	})
}

func (app *App) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:    app.cfg.HTTPAddr,
		Handler: app.Handler(),
	}
	errs := make(chan error, 1)
	go func() {
		errs <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownErr := server.Shutdown(context.Background())
		if shutdownErr != nil {
			return shutdownErr
		}
		return ctx.Err()
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

type emptyFeedProvider struct{}

func (emptyFeedProvider) List(_ int) []feedv1.FeedItem {
	return nil
}

type emptySearchProvider struct{}

func (emptySearchProvider) Search(_ parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	return nil, nil
}

type noopReactionRecorder struct{}

func (noopReactionRecorder) Record(_ userv1.UserReaction) error {
	return nil
}

package app

import (
	"context"
	"errors"
	"net/http"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	userv1 "github.com/hanq/articleflow/contracts/user/v1"
	feedclient "github.com/hanq/articleflow/services/gateway-api/internal/clients/feed"
	parserclient "github.com/hanq/articleflow/services/gateway-api/internal/clients/parser"
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
		FeedProvider:     feedclient.New(app.cfg.FeedServiceURL, http.DefaultClient),
		SearchProvider:   emptySearchProvider{},
		ParserJobClient:  parserclient.New(app.cfg.ParserServiceURL, http.DefaultClient),
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

type emptySearchProvider struct{}

func (emptySearchProvider) Search(_ parserv1.SearchQuery) ([]parserv1.ArticleCandidate, error) {
	return nil, nil
}

type noopReactionRecorder struct{}

func (noopReactionRecorder) Record(_ userv1.UserReaction) error {
	return nil
}

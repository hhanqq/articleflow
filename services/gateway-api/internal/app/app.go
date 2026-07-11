package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	articleclient "github.com/hanq/articleflow/services/gateway-api/internal/clients/article"
	feedclient "github.com/hanq/articleflow/services/gateway-api/internal/clients/feed"
	parserclient "github.com/hanq/articleflow/services/gateway-api/internal/clients/parser"
	"github.com/hanq/articleflow/services/gateway-api/internal/config"
	"github.com/hanq/articleflow/services/gateway-api/internal/reactions"
	httptransport "github.com/hanq/articleflow/services/gateway-api/internal/transport/http"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Handler() http.Handler {
	producer := articleflowkafka.NewWriterProducer(app.cfg.BrokerList())
	articles := articleclient.New(app.cfg.ArticleServiceURL, http.DefaultClient)
	parser := parserclient.New(app.cfg.ParserServiceURL, http.DefaultClient)
	return httptransport.NewRouter(httptransport.RouterDependencies{
		ServiceName:        app.cfg.ServiceName,
		ArticleProvider:    articles,
		FeedProvider:       feedclient.New(app.cfg.FeedServiceURL, http.DefaultClient),
		SearchProvider:     articles,
		ParserJobClient:    parser,
		ParserSourceClient: parser,
		ReactionRecorder:   reactions.NewPublisher(producer),
		RateLimitPerMinute: app.cfg.RateLimitPerMinute,
		FeedCacheTTL:       time.Duration(app.cfg.FeedCacheTTLSeconds) * time.Second,
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

package app

import (
	"context"
	"errors"
	"net/http"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/jobs"
	"github.com/hanq/articleflow/services/parser-service/internal/search"
	"github.com/hanq/articleflow/services/parser-service/internal/sources"
	httptransport "github.com/hanq/articleflow/services/parser-service/internal/transport/http"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Handler() http.Handler {
	store, closeStore, err := newJobStore(context.Background(), app.cfg, nil)
	if err != nil {
		panic(err)
	}
	_ = closeStore
	return app.handlerWithStore(store)
}

func (app *App) handlerWithStore(store jobs.Store) http.Handler {
	producer := articleflowkafka.NewWriterProducer(app.cfg.BrokerList())
	sourceRegistry := sources.BuildRegistry(app.cfg)
	searchUsecase := search.NewUsecase(producer, sourceRegistry.Parsers)
	manager := jobs.NewManager(store, searchUsecase)

	mux := http.NewServeMux()
	mux.Handle("/healthz", httptransport.NewHealthHandler(app.cfg.ServiceName))
	mux.Handle("/api/v1/parser/sources", httptransport.NewSourcesHandler(sourceRegistry.Sources))
	jobsHandler := httptransport.NewJobsHandler(manager)
	mux.Handle("/api/v1/parser/jobs", jobsHandler)
	mux.Handle("/api/v1/parser/jobs/", jobsHandler)
	return mux
}

func (app *App) Run(ctx context.Context) error {
	store, closeStore, err := newJobStore(ctx, app.cfg, nil)
	if err != nil {
		return err
	}
	defer closeStore()
	server := &http.Server{
		Addr:    app.cfg.HTTPAddr,
		Handler: app.handlerWithStore(store),
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

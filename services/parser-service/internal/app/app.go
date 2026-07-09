package app

import (
	"context"
	"errors"
	"net/http"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/packages/observability"
	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/jobs"
	"github.com/hanq/articleflow/services/parser-service/internal/scheduler"
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
	return app.handlerWithManager(app.newJobManager(store))
}

func (app *App) newJobManager(store jobs.Store) *jobs.Manager {
	producer := articleflowkafka.NewWriterProducer(app.cfg.BrokerList())
	sourceRegistry := sources.BuildRegistry(app.cfg)
	searchUsecase := search.NewUsecase(producer, sourceRegistry.Parsers)
	return jobs.NewManager(store, searchUsecase)
}

func (app *App) handlerWithManager(manager *jobs.Manager) http.Handler {
	sourceRegistry := sources.BuildRegistry(app.cfg)
	metrics := observability.NewMetricsRegistry()
	metrics.Inc("articleflow_service_info")

	mux := http.NewServeMux()
	mux.Handle("/healthz", httptransport.NewHealthHandler(app.cfg.ServiceName))
	mux.Handle("/metrics", observability.NewPrometheusHandler(app.cfg.ServiceName, metrics))
	mux.Handle("/api/v1/parser/sources", httptransport.NewSourcesHandler(sourceRegistry.Sources))
	jobsHandler := httptransport.NewJobsHandler(manager)
	mux.Handle("/api/v1/parser/jobs", jobsHandler)
	mux.Handle("/api/v1/parser/jobs/", jobsHandler)
	return observability.InstrumentHTTPRequests(metrics, mux)
}

func (app *App) Run(ctx context.Context) error {
	store, closeStore, err := newJobStore(ctx, app.cfg, nil)
	if err != nil {
		return err
	}
	defer closeStore()
	manager := app.newJobManager(store)
	server := &http.Server{
		Addr:    app.cfg.HTTPAddr,
		Handler: app.handlerWithManager(manager),
	}
	errs := make(chan error, 2)
	go func() {
		errs <- server.ListenAndServe()
	}()
	if app.cfg.SchedulerEnabled {
		go func() {
			errs <- scheduler.New(manager, app.cfg.SchedulerConfig()).Run(ctx)
		}()
	}

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

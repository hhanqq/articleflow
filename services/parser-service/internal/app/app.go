package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/jobs"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
	"github.com/hanq/articleflow/services/parser-service/internal/search"
	httptransport "github.com/hanq/articleflow/services/parser-service/internal/transport/http"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Handler() http.Handler {
	producer := articleflowkafka.NewWriterProducer(app.cfg.BrokerList())
	habrClient := habr.NewClient(habr.ClientOptions{
		BaseURL:     app.cfg.HabrBaseURL,
		MaxAttempts: app.cfg.HabrMaxAttempts,
		RetryDelay:  time.Duration(app.cfg.HabrRetryDelayMS) * time.Millisecond,
		Waiter:      habr.FixedDelayWaiter{Delay: time.Duration(app.cfg.HabrRequestDelayMS) * time.Millisecond},
	})
	searchUsecase := search.NewUsecase(producer, []search.Parser{habrClient})
	manager := jobs.NewManager(jobs.NewMemoryStore(), searchUsecase)

	mux := http.NewServeMux()
	jobsHandler := httptransport.NewJobsHandler(manager)
	mux.Handle("/api/v1/parser/jobs", jobsHandler)
	mux.Handle("/api/v1/parser/jobs/", jobsHandler)
	return mux
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

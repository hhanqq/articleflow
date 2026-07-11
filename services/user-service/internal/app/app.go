package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/packages/observability"
	"github.com/hanq/articleflow/services/user-service/internal/config"
	httptransport "github.com/hanq/articleflow/services/user-service/internal/transport/http"
	kafkatransport "github.com/hanq/articleflow/services/user-service/internal/transport/kafka"
	"github.com/hanq/articleflow/services/user-service/internal/usecase"
)

type App struct{ cfg config.Config }

func New(cfg config.Config) *App { return &App{cfg: cfg} }

func (app *App) handler(reactions usecase.ReactionStore) http.Handler {
	metrics := observability.NewMetricsRegistry()
	metrics.Inc("articleflow_service_info")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok"))
	})
	mux.Handle("/metrics", observability.NewPrometheusHandler(app.cfg.ServiceName, metrics))
	mux.Handle("/api/v1/users/", httptransport.NewUserReactionsHandler(reactions))
	return observability.InstrumentHTTPRequests(metrics, mux)
}

func (app *App) Run(ctx context.Context) error {
	reactions, closeStore, err := newReactionStore(ctx, app.cfg, openDatabase)
	if err != nil {
		return err
	}
	defer closeStore()
	server := &http.Server{
		Addr:    app.cfg.HTTPAddr,
		Handler: app.handler(reactions),
	}
	consumer := articleflowkafka.NewReaderConsumer(
		app.cfg.BrokerList(),
		app.cfg.UserReactionTopic,
		app.cfg.UserConsumerGroupID,
	)
	handler := kafkatransport.NewReactionHandler(reactions)
	fmt.Printf(
		"%s consuming %s from %v as %s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.UserReactionTopic,
		app.cfg.BrokerList(),
		app.cfg.UserConsumerGroupID,
		app.cfg.GRPCAddr,
	)
	errs := make(chan error, 2)
	go func() {
		errs <- server.ListenAndServe()
	}()
	go func() {
		errs <- consumeReactions(ctx, consumer, handler, app.cfg.UserConsumerMaxMessages)
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

func consumeReactions(ctx context.Context, consumer *articleflowkafka.ReaderConsumer, handler *kafkatransport.ReactionHandler, maxMessages int) error {
	defer consumer.Close()
	handled := 0
	for {
		if maxMessages > 0 && handled >= maxMessages {
			return nil
		}
		message, err := consumer.Fetch(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if err := handler.Handle(ctx, message); err != nil {
			return err
		}
		if err := consumer.Commit(ctx, message); err != nil {
			return err
		}
		handled++
	}
}

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/feed-service/internal/config"
	httptransport "github.com/hanq/articleflow/services/feed-service/internal/transport/http"
	kafkatransport "github.com/hanq/articleflow/services/feed-service/internal/transport/kafka"
	"github.com/hanq/articleflow/services/feed-service/internal/usecase"
)

type App struct{ cfg config.Config }

func New(cfg config.Config) *App { return &App{cfg: cfg} }

func (app *App) Handler() http.Handler {
	return newHTTPHandler(usecase.NewMemoryFeed())
}

func (app *App) Run(ctx context.Context) error {
	feed, closeStore, err := newFeedStore(ctx, app.cfg, openDatabase)
	if err != nil {
		return err
	}
	defer closeStore()
	server := &http.Server{
		Addr:    app.cfg.HTTPAddr,
		Handler: newHTTPHandler(feed),
	}
	consumer := articleflowkafka.NewReaderConsumer(
		app.cfg.BrokerList(),
		app.cfg.FeedScoredTopic,
		app.cfg.FeedConsumerGroupID,
	)
	handler := kafkatransport.NewScoredHandler(feed)
	errs := make(chan error, 2)
	go func() {
		errs <- server.ListenAndServe()
	}()
	go func() {
		errs <- consumeScored(ctx, consumer, handler, app.cfg.FeedConsumerMaxMessages)
	}()

	fmt.Printf(
		"%s consuming %s from %v as %s; HTTP on %s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.FeedScoredTopic,
		app.cfg.BrokerList(),
		app.cfg.FeedConsumerGroupID,
		app.cfg.HTTPAddr,
		app.cfg.GRPCAddr,
	)

	select {
	case <-ctx.Done():
		shutdownErr := server.Shutdown(context.Background())
		if shutdownErr != nil {
			return shutdownErr
		}
		_ = consumer.Close()
		return ctx.Err()
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newHTTPHandler(feed usecase.FeedStore) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", httptransport.NewHealthHandler("feed-service"))
	mux.Handle("/api/v1/feed", httptransport.NewFeedHandler(feed))
	return mux
}

func consumeScored(ctx context.Context, consumer *articleflowkafka.ReaderConsumer, handler *kafkatransport.ScoredHandler, maxMessages int) error {
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

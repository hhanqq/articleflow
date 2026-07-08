package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/config"
	"github.com/hanq/articleflow/services/article-service/internal/runtime"
	httptransport "github.com/hanq/articleflow/services/article-service/internal/transport/http"
	kafkatransport "github.com/hanq/articleflow/services/article-service/internal/transport/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/usecase"
)

type runtimeConsumer interface {
	runtime.MessageConsumer
	runtime.MessageCommitter
}

type App struct {
	cfg             config.Config
	databaseOpener  databaseOpener
	consumerFactory func(config.Config) runtimeConsumer
}

func New(cfg config.Config) *App {
	return &App{
		cfg:             cfg,
		databaseOpener:  openDatabase,
		consumerFactory: newKafkaConsumer,
	}
}

func (app *App) Run(ctx context.Context) error {
	store, closeStore, err := newArticleStore(ctx, app.cfg, app.databaseOpener)
	if err != nil {
		return err
	}
	defer closeStore()

	ingest := usecase.NewIngestUsecase(store)
	discovered := kafkatransport.NewDiscoveredHandler(ingest)
	handler := kafkatransport.NewRuntimeMessageHandler(discovered)
	consumer := app.consumerFactory(app.cfg)
	loop := runtime.NewConsumerLoop(consumer, handler, runtime.ConsumerLoopOptions{
		MaxMessages: app.cfg.ConsumerMaxMessages,
	})
	var server *http.Server
	errs := make(chan error, 2)
	if app.cfg.HTTPAddr != "" {
		server = &http.Server{
			Addr:    app.cfg.HTTPAddr,
			Handler: newHTTPHandler(app.cfg.ServiceName, store),
		}
		go func() {
			errs <- server.ListenAndServe()
		}()
	}
	go func() {
		errs <- loop.Run(ctx)
	}()

	fmt.Printf(
		"%s consuming %s from %v as %s; HTTP on %s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.BrokerList(),
		app.cfg.ArticleConsumerGroupID,
		app.cfg.HTTPAddr,
		app.cfg.GRPCAddr,
	)
	select {
	case <-ctx.Done():
		if server != nil {
			if err := server.Shutdown(context.Background()); err != nil {
				return err
			}
		}
		return ctx.Err()
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newKafkaConsumer(cfg config.Config) runtimeConsumer {
	return articleflowkafka.NewReaderConsumer(
		cfg.BrokerList(),
		cfg.ArticleDiscoveredTopic,
		cfg.ArticleConsumerGroupID,
	)
}

type articleHTTPStore interface {
	httptransport.ArticleReader
	httptransport.ArticleSearcher
}

func newHTTPHandler(serviceName string, reader articleHTTPStore) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", httptransport.NewHealthHandler(serviceName))
	mux.Handle("/api/v1/articles", httptransport.NewArticleHandler(reader))
	mux.Handle("/api/v1/articles/search", httptransport.NewArticleSearchHandler(reader))
	return mux
}

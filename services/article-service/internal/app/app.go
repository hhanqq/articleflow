package app

import (
	"context"
	"fmt"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/article-service/internal/config"
	"github.com/hanq/articleflow/services/article-service/internal/runtime"
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

	fmt.Printf(
		"%s consuming %s from %v as %s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.BrokerList(),
		app.cfg.ArticleConsumerGroupID,
		app.cfg.GRPCAddr,
	)
	return loop.Run(ctx)
}

func newKafkaConsumer(cfg config.Config) runtimeConsumer {
	return articleflowkafka.NewReaderConsumer(
		cfg.BrokerList(),
		cfg.ArticleDiscoveredTopic,
		cfg.ArticleConsumerGroupID,
	)
}

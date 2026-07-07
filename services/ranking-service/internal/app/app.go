package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/config"
	kafkatransport "github.com/hanq/articleflow/services/ranking-service/internal/transport/kafka"
	"github.com/hanq/articleflow/services/ranking-service/internal/usecase"
)

type App struct{ cfg config.Config }

func New(cfg config.Config) *App { return &App{cfg: cfg} }

func (app *App) Run(ctx context.Context) error {
	producer := articleflowkafka.NewWriterProducer(app.cfg.BrokerList())
	defer producer.Close()
	consumer := articleflowkafka.NewReaderConsumer(
		app.cfg.BrokerList(),
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.RankingConsumerGroupID,
	)
	handler := kafkatransport.NewDiscoveredHandler(usecase.NewRanker(), producer)
	fmt.Printf(
		"%s consuming %s from %v as %s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.BrokerList(),
		app.cfg.RankingConsumerGroupID,
		app.cfg.GRPCAddr,
	)
	return consumeDiscovered(ctx, consumer, handler, app.cfg.RankingConsumerMaxMessages)
}

func consumeDiscovered(ctx context.Context, consumer *articleflowkafka.ReaderConsumer, handler *kafkatransport.DiscoveredHandler, maxMessages int) error {
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

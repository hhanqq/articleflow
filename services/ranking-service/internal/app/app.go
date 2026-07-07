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
	discoveredConsumer := articleflowkafka.NewReaderConsumer(
		app.cfg.BrokerList(),
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.RankingConsumerGroupID,
	)
	reactionConsumer := articleflowkafka.NewReaderConsumer(
		app.cfg.BrokerList(),
		app.cfg.UserReactionTopic,
		app.cfg.ReactionConsumerGroupID,
	)
	signals := usecase.NewSignalStore()
	discoveredHandler := kafkatransport.NewDiscoveredHandler(usecase.NewRankerWithSignals(signals), producer)
	reactionHandler := kafkatransport.NewReactionHandler(signals)
	fmt.Printf(
		"%s consuming %s/%s from %v as %s/%s; gRPC planned on %s\n",
		app.cfg.ServiceName,
		app.cfg.ArticleDiscoveredTopic,
		app.cfg.UserReactionTopic,
		app.cfg.BrokerList(),
		app.cfg.RankingConsumerGroupID,
		app.cfg.ReactionConsumerGroupID,
		app.cfg.GRPCAddr,
	)
	errs := make(chan error, 2)
	go func() {
		errs <- consumeDiscovered(ctx, discoveredConsumer, discoveredHandler, app.cfg.RankingConsumerMaxMessages)
	}()
	go func() {
		errs <- consumeReactions(ctx, reactionConsumer, reactionHandler, app.cfg.ReactionConsumerMaxMessages)
	}()

	select {
	case <-ctx.Done():
		_ = discoveredConsumer.Close()
		_ = reactionConsumer.Close()
		return ctx.Err()
	case err := <-errs:
		return err
	}
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

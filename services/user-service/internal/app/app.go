package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/user-service/internal/config"
	kafkatransport "github.com/hanq/articleflow/services/user-service/internal/transport/kafka"
	"github.com/hanq/articleflow/services/user-service/internal/usecase"
)

type App struct{ cfg config.Config }

func New(cfg config.Config) *App { return &App{cfg: cfg} }

func (app *App) Run(ctx context.Context) error {
	reactions := usecase.NewMemoryReactions()
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
	return consumeReactions(ctx, consumer, handler, app.cfg.UserConsumerMaxMessages)
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

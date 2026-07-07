package runtime

import (
	"context"
	"errors"
	"io"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type MessageConsumer interface {
	Fetch(ctx context.Context) (articleflowkafka.Message, error)
	Close() error
}

type MessageHandler interface {
	Handle(ctx context.Context, message articleflowkafka.Message) error
}

type MessageCommitter interface {
	Commit(ctx context.Context, message articleflowkafka.Message) error
}

type ConsumerLoopOptions struct {
	MaxMessages int
}

type ConsumerLoop struct {
	consumer MessageConsumer
	handler  MessageHandler
	options  ConsumerLoopOptions
}

func NewConsumerLoop(consumer MessageConsumer, handler MessageHandler, options ConsumerLoopOptions) *ConsumerLoop {
	return &ConsumerLoop{
		consumer: consumer,
		handler:  handler,
		options:  options,
	}
}

func (loop *ConsumerLoop) Run(ctx context.Context) error {
	defer loop.consumer.Close()

	handled := 0
	for {
		if loop.options.MaxMessages > 0 && handled >= loop.options.MaxMessages {
			return nil
		}

		message, err := loop.consumer.Fetch(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if err := loop.handler.Handle(ctx, message); err != nil {
			return err
		}
		if committer, ok := loop.consumer.(MessageCommitter); ok {
			if err := committer.Commit(ctx, message); err != nil {
				return err
			}
		}
		handled++
	}
}

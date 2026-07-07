package kafka

import (
	"context"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type RuntimeMessageHandler struct {
	discovered *DiscoveredHandler
}

func NewRuntimeMessageHandler(discovered *DiscoveredHandler) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{discovered: discovered}
}

func (handler *RuntimeMessageHandler) Handle(ctx context.Context, message articleflowkafka.Message) error {
	_, err := handler.discovered.Handle(ctx, message)
	return err
}

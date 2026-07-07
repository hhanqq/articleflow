package runtime

import (
	"context"
	"io"
	"testing"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
)

type fakeConsumer struct {
	messages []articleflowkafka.Message
	commits  []string
	closed   bool
}

func (consumer *fakeConsumer) Fetch(_ context.Context) (articleflowkafka.Message, error) {
	if len(consumer.messages) == 0 {
		return articleflowkafka.Message{}, io.EOF
	}
	message := consumer.messages[0]
	consumer.messages = consumer.messages[1:]
	return message, nil
}

func (consumer *fakeConsumer) Close() error {
	consumer.closed = true
	return nil
}

func (consumer *fakeConsumer) Commit(_ context.Context, message articleflowkafka.Message) error {
	consumer.commits = append(consumer.commits, message.Key)
	return nil
}

type recordingHandler struct {
	keys []string
}

func (handler *recordingHandler) Handle(_ context.Context, message articleflowkafka.Message) error {
	handler.keys = append(handler.keys, message.Key)
	return nil
}

func TestConsumerLoopHandlesMessagesUntilMaxMessages(t *testing.T) {
	consumer := &fakeConsumer{
		messages: []articleflowkafka.Message{
			{Topic: "article.discovered.v1", Key: "article-1"},
			{Topic: "article.discovered.v1", Key: "article-2"},
		},
	}
	handler := &recordingHandler{}
	loop := NewConsumerLoop(consumer, handler, ConsumerLoopOptions{MaxMessages: 2})

	err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(handler.keys) != 2 {
		t.Fatalf("expected 2 handled messages, got %d", len(handler.keys))
	}
	if !consumer.closed {
		t.Fatal("expected consumer to be closed")
	}
	if len(consumer.commits) != 2 {
		t.Fatalf("expected 2 committed messages, got %d", len(consumer.commits))
	}
}

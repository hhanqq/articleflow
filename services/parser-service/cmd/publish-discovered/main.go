package main

import (
	"context"
	"log"

	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/devtools"
)

func main() {
	cfg := config.Load()
	producer := articleflowkafka.NewWriterProducer(cfg.BrokerList())
	defer producer.Close()

	if err := devtools.PublishSampleDiscovered(context.Background(), producer); err != nil {
		log.Fatal(err)
	}
	log.Printf("published sample article.discovered.v1 to Kafka brokers %v", cfg.BrokerList())
}

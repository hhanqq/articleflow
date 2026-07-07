package main

import (
	"context"
	"flag"
	"log"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
	articleflowkafka "github.com/hanq/articleflow/packages/kafka"
	"github.com/hanq/articleflow/services/parser-service/internal/config"
	"github.com/hanq/articleflow/services/parser-service/internal/parsers/habr"
	"github.com/hanq/articleflow/services/parser-service/internal/search"
)

func main() {
	queryText := flag.String("query", "", "Search query for Habr")
	limit := flag.Int("limit", 10, "Maximum number of articles to publish")
	flag.Parse()

	if *queryText == "" {
		log.Fatal("query flag is required")
	}

	cfg := config.Load()
	producer := articleflowkafka.NewWriterProducer(cfg.BrokerList())
	defer producer.Close()

	client := habr.NewClient(habr.ClientOptions{
		BaseURL:     cfg.HabrBaseURL,
		MaxAttempts: cfg.HabrMaxAttempts,
		RetryDelay:  time.Duration(cfg.HabrRetryDelayMS) * time.Millisecond,
		Waiter:      habr.FixedDelayWaiter{Delay: time.Duration(cfg.HabrRequestDelayMS) * time.Millisecond},
	})
	usecase := search.NewUsecase(producer, []search.Parser{client})
	candidates, err := usecase.SearchAndPublish(context.Background(), parserv1.SearchQuery{
		Text:    *queryText,
		Sources: []string{habr.SourceName},
		Limit:   *limit,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("published %d Habr article(s) for query %q", len(candidates), *queryText)
}

package main

import (
	"context"
	"log"

	"github.com/hanq/articleflow/services/feed-service/internal/app"
	"github.com/hanq/articleflow/services/feed-service/internal/config"
)

func main() {
	if err := app.New(config.Load()).Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}


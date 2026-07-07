package app

import (
	"context"
	"fmt"

	"github.com/hanq/articleflow/services/parser-service/internal/config"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App {
	return &App{cfg: cfg}
}

func (app *App) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		fmt.Printf("%s using Kafka %s\n", app.cfg.ServiceName, app.cfg.KafkaBrokers)
		return nil
	}
}


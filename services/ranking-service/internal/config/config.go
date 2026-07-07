package config

import sharedconfig "github.com/hanq/articleflow/packages/config"

type Config struct {
	ServiceName string
	GRPCAddr    string
}

func Load() Config {
	return Config{
		ServiceName: "ranking-service",
		GRPCAddr:    sharedconfig.String("RANKING_GRPC_ADDR", ":9004"),
	}
}


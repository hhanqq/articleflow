package config

import sharedconfig "github.com/hanq/articleflow/packages/config"

type Config struct {
	ServiceName string
	GRPCAddr    string
}

func Load() Config {
	return Config{
		ServiceName: "user-service",
		GRPCAddr:    sharedconfig.String("USER_GRPC_ADDR", ":9003"),
	}
}


package config

import sharedconfig "github.com/hanq/articleflow/packages/config"

type Config struct {
	ServiceName string
	GRPCAddr    string
}

func Load() Config {
	return Config{
		ServiceName: "feed-service",
		GRPCAddr:    sharedconfig.String("FEED_GRPC_ADDR", ":9002"),
	}
}


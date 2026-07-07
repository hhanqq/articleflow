package config

import sharedconfig "github.com/hanq/articleflow/packages/config"

type Config struct {
	ServiceName string
	HTTPAddr    string
}

func Load() Config {
	return Config{
		ServiceName: "gateway-api",
		HTTPAddr:    sharedconfig.String("GATEWAY_HTTP_ADDR", ":8080"),
	}
}


package config

import sharedconfig "github.com/hanq/articleflow/packages/config"

type Config struct {
	ServiceName      string
	HTTPAddr         string
	ParserServiceURL string
}

func Load() Config {
	return Config{
		ServiceName:      "gateway-api",
		HTTPAddr:         sharedconfig.String("GATEWAY_HTTP_ADDR", ":8080"),
		ParserServiceURL: sharedconfig.String("PARSER_SERVICE_URL", "http://localhost:8081"),
	}
}

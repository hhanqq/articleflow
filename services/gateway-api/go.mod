module github.com/hanq/articleflow/services/gateway-api

go 1.25.6

require (
	github.com/hanq/articleflow/packages/config v0.0.0
	github.com/hanq/articleflow/packages/kafka v0.0.0
	github.com/hanq/articleflow/packages/observability v0.0.0
)

require github.com/hanq/articleflow/contracts v0.0.0

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.48 // indirect
)

replace github.com/hanq/articleflow/packages/config => ../../packages/config

replace github.com/hanq/articleflow/contracts => ../../contracts

replace github.com/hanq/articleflow/packages/kafka => ../../packages/kafka

replace github.com/hanq/articleflow/packages/observability => ../../packages/observability

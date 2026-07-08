module github.com/hanq/articleflow/services/parser-service

go 1.25.6

require github.com/hanq/articleflow/packages/config v0.0.0

require github.com/hanq/articleflow/contracts v0.0.0

require (
	github.com/PuerkitoBio/goquery v1.10.0
	github.com/hanq/articleflow/packages/kafka v0.0.0
	github.com/jackc/pgx/v5 v5.10.0
)

require (
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.48 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace github.com/hanq/articleflow/packages/config => ../../packages/config

replace github.com/hanq/articleflow/contracts => ../../contracts

replace github.com/hanq/articleflow/packages/kafka => ../../packages/kafka

# feed-service

Owns ranked feed read models.

Runtime:

- consumes Kafka topic `feed.item.scored.v1`
- stores an in-memory ranked feed read model
- serves `GET /api/v1/feed?limit=20` over HTTP
- keeps the gRPC contract for the planned internal API

Run:

```bash
source ../../scripts/env.sh
KAFKA_BROKERS=localhost:9092 FEED_HTTP_ADDR=:8082 \
go run ./cmd/feed-service
```

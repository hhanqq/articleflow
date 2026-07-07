# ranking-service

Scores feed candidates and publishes ranking events.

Runtime:

- consumes Kafka topic `article.discovered.v1`
- scores article candidates
- publishes Kafka topic `feed.item.scored.v1`

Run:

```bash
source ../../scripts/env.sh
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/ranking-service
```

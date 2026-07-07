# ranking-service

Scores feed candidates and publishes ranking events.

Runtime:

- consumes Kafka topic `article.discovered.v1`
- consumes Kafka topic `user.reaction.created.v1`
- keeps in-memory tag signals from reactions for global feed personalization
- scores article candidates
- publishes Kafka topic `feed.item.scored.v1`

Run:

```bash
source ../../scripts/env.sh
KAFKA_BROKERS=localhost:9092 \
go run ./cmd/ranking-service
```

Useful env:

- `ARTICLE_DISCOVERED_TOPIC` defaults to `article.discovered.v1`
- `USER_REACTION_TOPIC` defaults to `user.reaction.created.v1`
- `RANKING_CONSUMER_GROUP_ID` defaults to `ranking-service`
- `RANKING_REACTION_CONSUMER_GROUP_ID` defaults to `ranking-service-reactions`

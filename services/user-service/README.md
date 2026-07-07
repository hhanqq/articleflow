# user-service

Owns users, interests, reactions, and saved articles.

Current runtime consumes `user.reaction.created.v1` from Kafka and stores reactions in memory.

Run:

```bash
go run ./cmd/user-service
```

Useful env:

- `KAFKA_BROKERS` defaults to `localhost:9092`
- `USER_REACTION_TOPIC` defaults to `user.reaction.created.v1`
- `USER_CONSUMER_GROUP_ID` defaults to `user-service`
- `USER_CONSUMER_MAX_MESSAGES` can limit messages for tests and one-shot runs

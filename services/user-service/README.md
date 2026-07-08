# user-service

Owns users, interests, reactions, and saved articles.

Current runtime consumes `user.reaction.created.v1` from Kafka and stores reactions in Postgres.

Run:

```bash
go run ./cmd/user-service
```

Useful env:

- `KAFKA_BROKERS` defaults to `127.0.0.1:9092`
- `USER_REACTION_TOPIC` defaults to `user.reaction.created.v1`
- `USER_CONSUMER_GROUP_ID` defaults to `user-service`
- `USER_CONSUMER_MAX_MESSAGES` can limit messages for tests and one-shot runs
- `USER_STORAGE_DRIVER` defaults to `postgres`; use `memory` only for isolated tests
- `USER_POSTGRES_DSN` defaults to local Docker Compose Postgres

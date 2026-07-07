# Articleflow Architecture

Articleflow is a microservice-first system for article discovery and personalized vertical feed delivery.

## Services

- `gateway-api`: public REST/JSON API for web/mobile clients.
- `parser-service`: fetches external sources and publishes discovered article events.
- `article-service`: owns article persistence and deduplication.
- `feed-service`: owns feed read models and pagination.
- `user-service`: owns users, interests, and reactions.
- `ranking-service`: scores article candidates.

## Communication

- Frontend to backend: REST/JSON through `gateway-api`.
- Internal request/response: gRPC with protobuf contracts.
- Internal asynchronous events: Kafka topics with versioned names.

## Event Flow

```text
parser-service -> article.discovered.v1 -> article-service
parser-service -> article.discovered.v1 -> ranking-service
ranking-service -> feed.item.scored.v1 -> feed-service
gateway-api -> feed-service HTTP -> frontend feed
gateway-api -> user-service gRPC -> user.reaction.created.v1
```

## Local Isolation

Go module downloads and tool binaries are stored inside project-local directories:

```text
.go/
.cache/
.bin/
```

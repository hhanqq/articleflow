# gateway-api

Public REST/JSON API for frontend clients.

Runtime:

- proxies parser jobs to parser-service
- proxies `GET /api/v1/feed` to feed-service
- publishes reaction requests to Kafka topic `user.reaction.created.v1`

Run:

```bash
source ../../scripts/env.sh
PARSER_SERVICE_URL=http://localhost:8081 FEED_SERVICE_URL=http://localhost:8082 \
go run ./cmd/gateway-api
```

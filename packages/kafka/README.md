# packages/kafka

Small Kafka adapter package used by services.

Responsibilities:

- shared `Message` type;
- `Producer` interface;
- in-memory producer for unit tests;
- JSON codec helpers for event payloads;
- `segmentio/kafka-go` writer and reader wrappers.

Runtime event topics are defined in `contracts/events/v1`.


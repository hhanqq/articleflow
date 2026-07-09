FROM golang:1.25.6-alpine AS build

ENV CGO_ENABLED=0
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3

FROM alpine:3.22

RUN adduser -D -H articleflow
USER articleflow

COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY deployments/postgres/migrations /migrations

ENTRYPOINT ["goose", "-dir", "/migrations"]

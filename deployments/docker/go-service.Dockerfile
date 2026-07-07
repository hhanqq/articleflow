FROM golang:1.25.6-alpine AS build

ARG SERVICE
WORKDIR /src

ENV GOPATH=/tmp/go
ENV GOMODCACHE=/tmp/go/pkg/mod
ENV GOCACHE=/tmp/go-build
ENV CGO_ENABLED=0

COPY . .
RUN test -n "$SERVICE"
RUN go build -trimpath -ldflags="-s -w" -o /out/articleflow-service "./services/${SERVICE}/cmd/${SERVICE}"

FROM alpine:3.22

ARG SERVICE
RUN adduser -D -H articleflow
USER articleflow

COPY --from=build /out/articleflow-service /usr/local/bin/articleflow-service
ENTRYPOINT ["/usr/local/bin/articleflow-service"]

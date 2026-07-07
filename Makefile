SHELL := /bin/bash
PROJECT_ROOT := $(CURDIR)

export GOPATH := $(PROJECT_ROOT)/.go
export GOMODCACHE := $(PROJECT_ROOT)/.go/pkg/mod
export GOCACHE := $(PROJECT_ROOT)/.cache/go-build
export GOBIN := $(PROJECT_ROOT)/.bin
export PATH := $(GOBIN):$(PATH)

.PHONY: env test tidy work-sync tools proto up down publish-sample-discovered e2e-article-chain

env:
	@mkdir -p "$(GOPATH)" "$(GOMODCACHE)" "$(GOCACHE)" "$(GOBIN)"
	@echo "GOPATH=$(GOPATH)"
	@echo "GOMODCACHE=$(GOMODCACHE)"
	@echo "GOCACHE=$(GOCACHE)"
	@echo "GOBIN=$(GOBIN)"

test: env
	go test ./contracts/... ./packages/config/... ./packages/kafka/... ./packages/observability/... ./services/article-service/... ./services/feed-service/... ./services/gateway-api/... ./services/parser-service/... ./services/ranking-service/... ./services/user-service/...

tidy: env
	go work sync
	cd contracts && go mod tidy
	cd packages/config && go mod tidy
	cd packages/kafka && go mod tidy
	cd packages/observability && go mod tidy
	cd services/article-service && go mod tidy
	cd services/feed-service && go mod tidy
	cd services/gateway-api && go mod tidy
	cd services/parser-service && go mod tidy
	cd services/ranking-service && go mod tidy
	cd services/user-service && go mod tidy

work-sync: env
	go work sync

tools: env
	go install github.com/bufbuild/buf/cmd/buf@v1.50.1
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

proto: env
	cd contracts && PATH="$(GOBIN):$$PATH" buf generate

up:
	docker compose -f deployments/docker-compose.yml up -d

down:
	docker compose -f deployments/docker-compose.yml down

publish-sample-discovered: env
	cd services/parser-service && go run ./cmd/publish-discovered

e2e-article-chain: env
	bash scripts/e2e_article_chain.sh

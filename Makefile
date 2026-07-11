SHELL := /bin/bash
PROJECT_ROOT := $(CURDIR)
APP_ENV_FILE ?= .env
DEFAULT_APP_ENV_FILE := deployments/env/local.env.example
COMPOSE_APP_ENV_FILE := $(if $(wildcard $(APP_ENV_FILE)),$(APP_ENV_FILE),$(DEFAULT_APP_ENV_FILE))

export GOPATH := $(PROJECT_ROOT)/.go
export GOMODCACHE := $(PROJECT_ROOT)/.go/pkg/mod
export GOCACHE := $(PROJECT_ROOT)/.cache/go-build
export GOBIN := $(PROJECT_ROOT)/.bin
export PATH := $(GOBIN):$(PATH)

.PHONY: env env-init test web-test ci compose-config migrate-up app-up app-down web dev-start dev-stop dev-status dev-smoke tidy work-sync tools proto up down publish-sample-discovered search-habr e2e-article-chain e2e-ranking-feed e2e-personalized-feed load-feed-smoke web-e2e

env:
	@mkdir -p "$(GOPATH)" "$(GOMODCACHE)" "$(GOCACHE)" "$(GOBIN)"
	@echo "GOPATH=$(GOPATH)"
	@echo "GOMODCACHE=$(GOMODCACHE)"
	@echo "GOCACHE=$(GOCACHE)"
	@echo "GOBIN=$(GOBIN)"

env-init:
	@test -f "$(APP_ENV_FILE)" || cp "$(DEFAULT_APP_ENV_FILE)" "$(APP_ENV_FILE)"
	@echo "env file: $(APP_ENV_FILE)"

test: env
	go test ./contracts/... ./packages/config/... ./packages/kafka/... ./packages/observability/... ./services/article-service/... ./services/feed-service/... ./services/gateway-api/... ./services/parser-service/... ./services/ranking-service/... ./services/user-service/...

web-test:
	cd web && npm test

ci: test web-test
	docker compose -f deployments/docker-compose.yml config >/dev/null
	docker compose --env-file deployments/env/local.env.example -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml config >/dev/null
	docker compose --env-file deployments/env/stage.env.example -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml config >/dev/null
	docker compose --env-file deployments/env/prod.env.example -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml config >/dev/null
	git diff --check

compose-config:
	docker compose --env-file "$(COMPOSE_APP_ENV_FILE)" -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml config

migrate-up:
	docker compose --env-file "$(COMPOSE_APP_ENV_FILE)" -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml run --rm migrations

app-up:
	docker compose --env-file "$(COMPOSE_APP_ENV_FILE)" -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml up -d --build

app-down:
	docker compose --env-file "$(COMPOSE_APP_ENV_FILE)" -f deployments/docker-compose.yml -f deployments/docker-compose.app.yml down

web:
	cd web && npm run start

dev-start: env
	bash scripts/dev_start.sh

dev-stop:
	bash scripts/dev_stop.sh

dev-status:
	bash scripts/dev_status.sh

dev-smoke:
	bash scripts/dev_smoke.sh

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

search-habr: env
	cd services/parser-service && go run ./cmd/search-habr --query "$${QUERY:-go kafka}" --limit "$${LIMIT:-10}"

e2e-article-chain: env
	bash scripts/e2e_article_chain.sh

e2e-ranking-feed: env
	bash scripts/e2e_ranking_feed.sh

e2e-personalized-feed:
	bash scripts/e2e_personalized_feed.sh

load-feed-smoke:
	bash scripts/load_feed_smoke.sh

web-e2e:
	cd web && npm run e2e

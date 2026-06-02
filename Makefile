SHELL := /bin/bash
BINARY := bin/lipd-server
COMPOSE := deployments/docker-compose/docker-compose.yml
DATABASE_URL ?= postgres://yugabyte:yugabyte@localhost:5433/yugabyte

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: db migrate certs ## Full local dev environment (db + schema + certs)

.PHONY: db
db: ## Start YugabyteDB via docker-compose
	docker compose -f $(COMPOSE) up -d yugabyte

.PHONY: migrate
migrate: ## Apply all SQL migrations
	@for f in migrations/*.sql; do \
	  echo "applying $$f"; \
	  psql "$(DATABASE_URL)" -f "$$f"; \
	done

.PHONY: certs
certs: ## Generate dev mTLS certificates into ./tls
	go run ./cmd/gen-certs -out ./tls

.PHONY: build
build: ## Build the server binary
	go build -o $(BINARY) ./cmd/lipd-server

.PHONY: run
run: build ## Build and run the server (mTLS on :8443)
	DATABASE_URL="$(DATABASE_URL)" ./$(BINARY)

.PHONY: test
test: ## Run tests with coverage
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

.PHONY: lint
lint: ## Run go vet + gofmt check
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

.PHONY: docker-build
docker-build: ## Build the production container image
	docker build -f deployments/docker/Dockerfile -t routefast-ee/lipd-server:dev .

.PHONY: clean
clean: ## Remove build artifacts and dev containers
	rm -rf bin/ coverage.out coverage.html tls/
	docker compose -f $(COMPOSE) down -v

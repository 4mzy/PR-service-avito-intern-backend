.PHONY: run build test lint migrate clean docker-up docker-down

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable

build:
	go build -o bin/server ./cmd/server

run: migrate
	DATABASE_URL=$(DATABASE_URL) PORT=8080 go run ./cmd/server

test:
	go test -v ./...

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

migrate:
	@which migrate > /dev/null || (echo "migrate tool not found. Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@which migrate > /dev/null || (echo "migrate tool not found. Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@which migrate > /dev/null || (echo "migrate tool not found. Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

clean:
	rm -rf bin/

DOCKER_COMPOSE_CMD := $(shell which docker-compose 2>/dev/null)
ifeq ($(DOCKER_COMPOSE_CMD),)
	DOCKER_COMPOSE_CMD := docker compose
endif

docker-up:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "Error: Docker is not installed. Please install Docker Desktop from https://www.docker.com/products/docker-desktop/"; \
		exit 1; \
	fi
	@if ! docker info >/dev/null 2>&1; then \
		echo "Error: Docker daemon is not running."; \
		echo "Please start Docker Desktop or Docker daemon and try again."; \
		exit 1; \
	fi
	@if command -v docker-compose >/dev/null 2>&1; then \
		docker-compose up -d; \
	elif docker compose version >/dev/null 2>&1; then \
		docker compose up -d; \
	else \
		echo "Error: Docker Compose is not installed."; \
		echo "Install via: brew install docker-compose"; \
		exit 1; \
	fi

docker-down:
	@if command -v docker-compose >/dev/null 2>&1; then \
		docker-compose down; \
	elif docker compose version >/dev/null 2>&1; then \
		docker compose down; \
	fi

docker-down-volumes:
	@if command -v docker-compose >/dev/null 2>&1; then \
		docker-compose down -v; \
	elif docker compose version >/dev/null 2>&1; then \
		docker compose down -v; \
	fi

docker-logs:
	@if command -v docker-compose >/dev/null 2>&1; then \
		docker-compose logs -f server; \
	elif docker compose version >/dev/null 2>&1; then \
		docker compose logs -f server; \
	fi


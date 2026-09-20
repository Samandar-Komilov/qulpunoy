-include .env

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= orderconc
DB_SSLMODE ?= disable

DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
MIGRATIONS_DIR ?= migrations

.PHONY: run build tidy test test-concurrency test-down migrate-up migrate-down migrate-status compose-up compose-down

run:
	go run ./cmd/server

test:
	docker compose -f docker-compose.test.yml up -d --wait db redis
	DB_HOST=localhost DB_PORT=5436 REDIS_HOST=localhost REDIS_PORT=6381 \
		go test ./test/integration -count=1 -v

test-concurrency:
	docker compose -f docker-compose.test.yml up -d --wait db redis
	DB_HOST=localhost DB_PORT=5436 REDIS_HOST=localhost REDIS_PORT=6381 \
		go test ./test/integration -count=1 -v -race -run 'TestOrderAPI/TestConcurrent'

test-down:
	docker compose stop db redis

build:
	go build -o bin/qulpunoy ./cmd/server

tidy:
	go mod tidy

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down -v
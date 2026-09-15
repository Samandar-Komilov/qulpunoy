-include .env

.PHONY: run build tidy

run:
	go run ./cmd/server

build:
	go build -o bin/qulpunoy ./cmd/server

tidy:
	go mod tidy
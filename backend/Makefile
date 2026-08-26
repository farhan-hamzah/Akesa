include .env
export

MIGRATE := $(shell which migrate)

.PHONY: help db-up db-down migrate-up migrate-down migrate-create run build fmt tidy

help:
	@echo "Akesa backend - available commands:"
	@echo "  make db-up            start local Postgres (docker compose)"
	@echo "  make db-down          stop local Postgres"
	@echo "  make migrate-up       apply all pending migrations"
	@echo "  make migrate-down     roll back the last migration"
	@echo "  make migrate-create name=create_something   scaffold a new migration pair"
	@echo "  make run              run the API locally"
	@echo "  make build            build the API binary into ./bin"
	@echo "  make fmt              gofmt every .go file"
	@echo "  make tidy             go mod tidy"

db-up:
	docker compose up -d

db-down:
	docker compose down

migrate-up:
	@if [ -z "$(MIGRATE)" ]; then echo "install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@if [ -z "$(MIGRATE)" ]; then echo "install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@if [ -z "$(name)" ]; then echo "usage: make migrate-create name=create_something"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(name)

run:
	go run ./cmd/api

build:
	go build -o bin/akesa-api ./cmd/api

fmt:
	gofmt -w .

tidy:
	go mod tidy

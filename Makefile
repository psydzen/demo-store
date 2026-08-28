DATABASE_URL ?= postgres://quiz:quiz@localhost:5433/quiz?sslmode=disable

.PHONY: up down run test lint

# up starts Postgres and the app in Docker.
up:
	docker compose up --build

# down stops everything and drops the database volume.
down:
	docker compose down -v

# run starts only Postgres in Docker and the app on the host.
run:
	docker compose up -d db
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/quiz

test:
	go test ./...

lint:
	gofmt -l .
	go vet ./...

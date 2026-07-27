.PHONY: test test-cover test-docker build up

test:
	go test ./... -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-docker:
	docker run --rm -v $(PWD):/app -w /app golang:1.23-alpine sh -c "go mod download && go test ./... -count=1"

build:
	docker compose build

up:
	docker compose up --build

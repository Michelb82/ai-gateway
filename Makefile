.PHONY: test test-cover test-docker test-integration build up

test:
	go test ./... -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-docker:
	docker run --rm -v $(PWD):/app -w /app golang:1.23-alpine sh -c "go mod download && go test ./... -count=1"

test-integration:
	docker run --rm --network dev \
		-e REDIS_ADDR=redis:6379 \
		-v $(PWD):/app -w /app golang:1.23-alpine \
		sh -c "go mod download && go test ./internal/queue/ -tags=integration -count=1 -v"

build:
	docker compose build

up:
	docker compose up --build

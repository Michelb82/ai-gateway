# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ai-gateway ./cmd/gateway

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache libcap \
  && addgroup -S app \
  && adduser -S app -G app

WORKDIR /app
COPY --from=builder /ai-gateway /app/ai-gateway
RUN setcap 'cap_net_bind_service=+ep' /app/ai-gateway

USER app

EXPOSE 80

ENTRYPOINT ["/app/ai-gateway"]

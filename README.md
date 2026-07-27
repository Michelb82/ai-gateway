# Construction AI Gateway

Standalone Go service that consumes CloudEvent-shaped JSON from Redis, calls Ollama, and publishes responses back to Redis.

## Architecture

- Input queue: `queue:ai.requests`
- Output queue: `queue:ai.responses`
- Ollama endpoint: `http://foundation-model:11434/v1/chat/completions`
- Docker network: `construction_dev` (external)

## Prerequisites

1. Start the construction stack so Redis and Ollama are available:

```bash
cd /home/michel/Project/construction
docker compose up -d redis
docker compose -f docker-compose.ollama.yml up -d
```

2. Ensure the `construction_dev` network exists:

```bash
docker network ls | grep construction_dev
```

## Run locally in Docker

```bash
cd /home/michel/Project/construction-ai-gateway
docker compose up --build
```

## Configuration

Copy `.env.example` and adjust as needed:

| Variable | Default |
|----------|---------|
| `REDIS_ADDR` | `redis:6379` |
| `INPUT_QUEUE` | `ai.requests` |
| `OUTPUT_QUEUE` | `ai.responses` |
| `OLLAMA_URL` | `http://foundation-model:11434` |
| `OLLAMA_MODEL` | `qwen3:14b-q4_K_M` |
| `BRPOP_TIMEOUT` | `5` |

## Manual smoke test

With the gateway running:

```bash
chmod +x scripts/push-test-event.sh
./scripts/push-test-event.sh
```

Or manually:

```bash
docker exec redis_construction redis-cli LPUSH queue:ai.requests '{"type":"com.buildright.ai.chat","source":"/ai-gateway","id":"test-1","organisation_id":"7","time":"2026-07-27T14:30:00+00:00","datacontenttype":"application/json","data":{"system_prompt":"You are helpful.","prompt":"Say pong"}}'

docker exec redis_construction redis-cli BRPOP queue:ai.responses 30
```

## Tests

Run unit tests with Docker (no local Go required):

```bash
make test-docker
```

With Go installed locally:

```bash
make test
make test-cover
```

## Message format

Input and output messages follow the construction CloudEvent derivative documented in `feature/plan/ai_gateway_dummy_setup.md`.

Successful responses use `com.buildright.ai.chat.completed` with `subject` set to the request `id`. Failures use `com.buildright.ai.chat.failed` with `data.error`.

# Construction AI Gateway

Standalone Go service that exposes AI capabilities over Redis CloudEvents and keeps Ollama model details inside the gateway.

## Architecture

- Input queue: `queue:ai.requests`
- Output queue: `queue:ai.responses`
- Capabilities: `routing`, `intent-classification`, `translate`
- Ollama chat: `http://foundation-model:11434/v1/chat/completions`
- Health: `GET /health` (HTML), `GET /health.json` (JSON) on port 80
- Docker network: `construction_dev` (external)

Applications request a **capability**. The gateway maps that capability to a configured (quantized) model and never accepts caller-supplied model names.

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

3. Pull the models configured for each capability (defaults below).

## Run locally in Docker

```bash
cd /home/michel/Project/construction-ai-gateway
docker compose up --build
```

Health endpoints are published on host port `18080` by default (`http://localhost:18080/health` and `/health.json`).

## Configuration

Copy `.env.example` and adjust as needed:

| Variable | Default |
|----------|---------|
| `REDIS_ADDR` | `redis:6379` |
| `INPUT_QUEUE` | `ai.requests` |
| `OUTPUT_QUEUE` | `ai.responses` |
| `OLLAMA_URL` | `http://foundation-model:11434` |
| `OLLAMA_MODEL_ROUTING` | `qwen3:1.7b-q4_K_M` |
| `OLLAMA_MODEL_INTENT` | `qwen3:4b-q4_K_M` |
| `OLLAMA_MODEL_TRANSLATE` | `qwen3:14b-q4_K_M` |
| `OLLAMA_MODEL_ROUTING_TTL` | `5m` |
| `OLLAMA_MODEL_INTENT_TTL` | `5m` |
| `OLLAMA_MODEL_TRANSLATE_TTL` | `2m` |
| `HTTP_ADDR` | `:80` |
| `BRPOP_TIMEOUT` | `5` |
| `DEBUG` | `false` |

Each `*_TTL` value is passed to Ollama as `keep_alive` for that capability’s model (for example `2m`, `90s`).

## Capability contract

Request type: `com.buildright.ai.request`

```json
{
  "type": "com.buildright.ai.request",
  "source": "/some-service",
  "id": "abc-123",
  "organisation_id": "7",
  "time": "2026-07-28T22:00:00+00:00",
  "datacontenttype": "application/json",
  "data": {
    "capability": "intent-classification",
    "input": { "message": "I need my living room painted" }
  }
}
```

| Capability | Input | Result shape |
|------------|--------|--------------|
| `routing` | `message` | `{ "capability": "<next-capability>" }` |
| `intent-classification` | `message` | `{ "intent": "...", "confidence": 0.0-1.0 }` |
| `translate` | `text`, `source_locale`, `target_locale` | `{ "text": "..." }` |

Translate example:

```json
{
  "data": {
    "capability": "translate",
    "input": {
      "text": "Elektrische installaties voor woningen",
      "source_locale": "nl",
      "target_locale": "en"
    }
  }
}
```

Success: `com.buildright.ai.request.completed` with request `data` merged back in, plus `data.capability` and `data.result` (no `model` field). Gateway fields overwrite matching request keys.

Failure: `com.buildright.ai.request.failed` with request `data` merged back in, plus `data.error` (and `data.capability` when known). Unavailable models are logged and returned this way.

## Health endpoints

| Path | Format | Meaning |
|------|--------|---------|
| `/health` | HTML | Per-capability readiness page |
| `/health.json` | JSON | Same data for probes/automation |

Status values:

- Per capability: `available` / `unavailable`
- Overall: `ready` / `not_ready`

HTTP status is `200` when ready and `503` when not ready. Model names appear only on health endpoints (ops), not on successful queue responses.

## Manual smoke test

With the gateway running:

```bash
chmod +x scripts/push-test-event.sh
./scripts/push-test-event.sh
CAPABILITY=routing ./scripts/push-test-event.sh
```

```bash
curl -sS http://localhost:18080/health.json
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

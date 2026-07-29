# Construction AI Gateway

Standalone Go service that exposes AI capabilities over Redis CloudEvents and keeps Ollama model details inside the gateway.

## Architecture

- Input queue: `queue:ai.requests` (producers still `LPUSH` here)
- Internal priority lanes: `queue:ai.requests:critical|high|medium|low`
- Output queue: `queue:ai.responses`
- Capabilities: `routing`, `intent-classification`, `translate`
- Ollama chat: `http://llm-model:11434/v1/chat/completions`
- Health: `GET /health` (HTML), `GET /health.json` (JSON) on port 80
- Docker network: `construction_dev` (external)

Applications request a **capability**. The gateway maps that capability to a configured (quantized) model and never accepts caller-supplied model names.

Optional `data.priority` on each request selects a processing lane: `CRITICAL`, `HIGH`, `MEDIUM`, or `LOW` (case-insensitive). Missing or invalid values default to `LOW`. The gateway demuxes the input list into Redis lanes and schedules work with fairness counters (see Configuration).

## Prerequisites

1. Start the sibling construction stack so Redis and Ollama (`llm-model`) are available on the shared Docker network.

2. Ensure the `construction_dev` network exists:

```bash
docker network ls | grep construction_dev
```

3. Pull the models configured for each capability (defaults below).

## Run locally in Docker

```bash
cp .env.dist .env
# Edit .env if your local stack uses different hostnames or CloudEvent type prefixes.
docker compose up --build
```

Health endpoints are published on host port `18080` by default (`http://localhost:18080/health` and `/health.json`).

## Configuration

Copy `.env.dist` to `.env` and adjust as needed. Public defaults use placeholder names (`llm-model`, `com.mywebsite.ai`); override them in `.env` for your private deployment.

| Variable | Default |
|----------|---------|
| `REDIS_ADDR` | `redis:6379` |
| `INPUT_QUEUE` | `ai.requests` |
| `OUTPUT_QUEUE` | `ai.responses` |
| `OLLAMA_URL` | `http://llm-model:11434` |
| `OLLAMA_MODEL_ROUTING` | `qwen3:1.7b-q4_K_M` |
| `OLLAMA_MODEL_INTENT` | `qwen3:4b-q4_K_M` |
| `OLLAMA_MODEL_TRANSLATE` | `qwen3:14b-q4_K_M` |
| `OLLAMA_MODEL_ROUTING_TTL` | `5m` |
| `OLLAMA_MODEL_INTENT_TTL` | `5m` |
| `OLLAMA_MODEL_TRANSLATE_TTL` | `2m` |
| `CLOUDEVENT_TYPE_PREFIX` | `com.mywebsite.ai` |
| `HTTP_ADDR` | `:80` |
| `BRPOP_TIMEOUT` | `5` |
| `PRIORITY_HIGH_COUNT` | `3` |
| `PRIORITY_MEDIUM_COUNT` | `3` |
| `DEBUG` | `false` |

Each `*_TTL` value is passed to Ollama as `keep_alive` for that capability’s model (for example `2m`, `90s`).

Request/completed/failed CloudEvent types are derived from `CLOUDEVENT_TYPE_PREFIX` as `{prefix}.request`, `{prefix}.request.completed`, and `{prefix}.request.failed`.

### Priority fairness

Scheduling order:

1. `CRITICAL` always first (does not affect counters)
2. `LOW` when due (see below)
3. `MEDIUM` when due (`PRIORITY_HIGH_COUNT` HIGH messages since the last due MEDIUM)
4. otherwise prefer `HIGH`, then `MEDIUM`, then `LOW`

Fairness counters reset when their threshold is reached:

- Every `PRIORITY_HIGH_COUNT` HIGH messages unlocks one MEDIUM
- Every `PRIORITY_MEDIUM_COUNT` MEDIUM messages unlocks one LOW
- Every `PRIORITY_HIGH_COUNT * PRIORITY_MEDIUM_COUNT` HIGH messages unlocks one LOW

With the defaults (`3` / `3`): every 3 HIGH → 1 MEDIUM; every 3 MEDIUM → 1 LOW; every 9 HIGH → 1 LOW. Example: after 9 HIGH and 2 MEDIUM, a LOW may run; processing one more MEDIUM then unlocks another LOW.

## Capability contract

Request type: `{CLOUDEVENT_TYPE_PREFIX}.request` (default `com.mywebsite.ai.request`)

```json
{
  "type": "com.mywebsite.ai.request",
  "source": "/some-service",
  "id": "abc-123",
  "organisation_id": "7",
  "time": "2026-07-28T22:00:00+00:00",
  "datacontenttype": "application/json",
  "data": {
    "capability": "intent-classification",
    "priority": "HIGH",
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

Success: `{prefix}.request.completed` with request `data` merged back in, plus `data.capability` and `data.result` (no `model` field). Gateway fields overwrite matching request keys.

Failure: `{prefix}.request.failed` with request `data` merged back in, plus `data.error` (and `data.capability` when known). Unavailable models are logged and returned this way.

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

If your `.env` overrides `CLOUDEVENT_TYPE_PREFIX`, export the same value (or `EVENT_TYPE`) when running the smoke script.

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

Dev-only integration test against live Redis on `construction_dev` (not run by `make test` / `make test-docker`):

```bash
make test-integration
```

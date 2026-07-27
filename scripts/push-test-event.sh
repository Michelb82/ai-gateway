#!/usr/bin/env bash
set -euo pipefail

REDIS_CONTAINER="${REDIS_CONTAINER:-redis_construction}"
INPUT_QUEUE="${INPUT_QUEUE:-queue:ai.requests}"
OUTPUT_QUEUE="${OUTPUT_QUEUE:-queue:ai.responses}"
TIMEOUT="${TIMEOUT:-30}"

PAYLOAD='{
  "type": "com.buildright.ai.chat",
  "source": "/ai-gateway",
  "subject": null,
  "id": "smoke-test-1",
  "organisation_id": "7",
  "time": "2026-07-27T14:30:00+00:00",
  "datacontenttype": "application/json",
  "data": {
    "system_prompt": "You are a helpful assistant.",
    "prompt": "Reply with exactly: pong",
    "model": "qwen3:14b-q4_K_M"
  }
}'

echo "Pushing test event to ${INPUT_QUEUE}..."
docker exec "${REDIS_CONTAINER}" redis-cli LPUSH "${INPUT_QUEUE}" "${PAYLOAD}" >/dev/null

echo "Waiting for response on ${OUTPUT_QUEUE} (timeout ${TIMEOUT}s)..."
docker exec "${REDIS_CONTAINER}" redis-cli BRPOP "${OUTPUT_QUEUE}" "${TIMEOUT}"

#!/usr/bin/env bash
set -euo pipefail

REDIS_CONTAINER="${REDIS_CONTAINER:-redis_construction}"
INPUT_QUEUE="${INPUT_QUEUE:-queue:ai.requests}"
OUTPUT_QUEUE="${OUTPUT_QUEUE:-queue:ai.responses}"
TIMEOUT="${TIMEOUT:-30}"
CAPABILITY="${CAPABILITY:-intent-classification}"
MESSAGE="${MESSAGE:-I need my living room painted}"
EVENT_TYPE="${EVENT_TYPE:-${CLOUDEVENT_TYPE_PREFIX:-com.mywebsite.ai}.request}"

PAYLOAD=$(cat <<EOF
{
  "type": "${EVENT_TYPE}",
  "source": "/ai-gateway",
  "subject": null,
  "id": "smoke-test-1",
  "organisation_id": "7",
  "time": "2026-07-28T22:00:00+00:00",
  "datacontenttype": "application/json",
  "data": {
    "capability": "${CAPABILITY}",
    "input": {
      "message": "${MESSAGE}"
    }
  }
}
EOF
)

echo "Pushing ${CAPABILITY} test event to ${INPUT_QUEUE}..."
docker exec "${REDIS_CONTAINER}" redis-cli LPUSH "${INPUT_QUEUE}" "${PAYLOAD}" >/dev/null

echo "Waiting for response on ${OUTPUT_QUEUE} (timeout ${TIMEOUT}s)..."
docker exec "${REDIS_CONTAINER}" redis-cli BRPOP "${OUTPUT_QUEUE}" "${TIMEOUT}"

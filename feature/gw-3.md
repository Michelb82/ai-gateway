# GW-3: Make data-plane lifecycle resilient and transactional

## Problem

Two related failure modes:

1. One bad Redis payload makes `Consume` error and the worker exits; it is not auto-restarted.
2. `Apply` can swap the registry (and Redis) before `EnsureModels` succeeds; on failure the manager keeps the old snapshot while the live data plane has already moved.

## Impact

- Single poisoned queue message can stop all consumption until process restart
- Control-plane “current snapshot” can disagree with the live registry / Redis client
- Health and workers may observe partial / inconsistent apply state

## Shorthand solution

- Skip or dead-letter malformed queue messages instead of killing the worker
- Auto-recover / restart the worker on unexpected exit
- Make apply transactional: prepare → verify models → atomic swap, or roll back registry/Redis on failure

## Related code

- `internal/queue/redis.go` (parse errors returned to `Consume`)
- `internal/worker/worker.go` (`Run` exits on consume error)
- `cmd/gateway/main.go` (`dataPlane.Apply` order: registry/Redis before `EnsureModels`)
- `internal/configmgmt/manager.go` (keeps previous snapshot when `onApply` fails)

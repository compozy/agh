# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `internal/bridgesdk` as the shared provider runtime substrate for provider-scoped bridge adapters.
- Cover runtime boot, typed Host API access, instance-cache synchronization, ingress guards, adapter-local dedup, optional batching, error classification/recovery, and lifecycle helpers.

## Important Decisions
- Treat the PRD/techspec/ADRs as the approved design for this task; do not reopen design work.
- Keep task scope on the shared substrate and validate it with new package-level integration tests instead of fully migrating the Telegram reference adapter in this task.
- Prefer bridge-specific helpers over overly generic abstractions where the bridge contract already supplies stable types (`BridgeInstance`, `InboundMessageEnvelope`, `DeliveryRequest`, `DeliveryAck`).
- Preserve bound secret material inside the managed instance cache across Host API resyncs because provider-scoped list/get responses only carry bridge instance state.
- Keep ingress hardening and runtime hooks provider-neutral by exposing explicit handler interfaces and seams instead of embedding platform-specific assumptions in the shared package.

## Learnings
- The current provider-scoped handshake includes managed instances plus bound secrets, but the new Host API list/get surfaces only return `BridgeInstance` state; cache synchronization must preserve launch-time secret material separately.
- The only existing bridge-adapter runtime substrate is the embedded JSON-RPC peer and lifecycle logic inside `sdk/examples/telegram-reference/main.go`.
- `make verify`, focused package coverage, and `go test -tags integration ./internal/bridgesdk` all pass with the new package, so the shared substrate can land without first migrating a concrete provider onto it.

## Files / Surfaces
- `internal/subprocess/handshake.go`
- `internal/extension/protocol/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/host_api_bridges.go`
- `internal/extensiontest/bridge_adapter_harness.go`
- `sdk/examples/telegram-reference/main.go`
- `.resources/openclaw/src/plugin-sdk/webhook-ingress.ts`
- `.resources/hermes/gateway/platforms/helpers.py`
- `.resources/goclaw/internal/providers/retry.go`
- `internal/bridgesdk/cache.go`
- `internal/bridgesdk/runtime.go`
- `internal/bridgesdk/webhook.go`
- `internal/bridgesdk/dedup.go`
- `internal/bridgesdk/batching.go`
- `internal/bridgesdk/errors.go`
- `internal/bridgesdk/hostapi.go`
- `internal/bridgesdk/peer.go`
- `internal/bridgesdk/*_test.go`

## Errors / Corrections
- Fixed `errcheck` in webhook body handling by explicitly closing the guarded body reader.
- Fixed `lostcancel` in the batcher constructor and normalized test socket cleanup to keep lint clean under `make verify`.

## Ready for Next Run
- Task implementation and verification are complete; only task tracking and scoped commit creation remain.

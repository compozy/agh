# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add per-instance channel observability/health reporting across `internal/observe`, the daemon-owned channel runtime, and the shared channel/health APIs without regressing existing daemon/session health fields.
- Verification target met: `make verify` passed, with task-surface coverage at `internal/channels` 80.5%, `internal/observe` 82.2%, `internal/api/core` 80.5% (integration `-coverpkg`), and `internal/api/httpapi` 82.1%.

## Important Decisions
- Aggregate channel health in `internal/observe` from three daemon-owned sources: persisted channel instances/routes, broker delivery metrics, and live auth/runtime issue signals.
- Keep transport payloads additive by returning nested health data (`health` / `channel_health`) instead of mutating the persisted `ChannelInstance` schema.
- Preserve terminal delivery errors as the operator-visible `last_error` signal even when later resume retries hit generic transport failures.

## Learnings
- HTTP integration fixtures are simpler with global-scope channel instances unless the test explicitly needs a real workspace ID.
- OpenAPI/SDK regeneration changed the daemon health contract shape for web consumers; frontend fixture/type tests must include the additive `channels` block.

## Files / Surfaces
- `internal/observe/channels.go`
- `internal/observe/health.go`
- `internal/observe/channels_test.go`
- `internal/channels/delivery_metrics.go`
- `internal/channels/delivery_broker.go`
- `internal/channels/delivery_broker_test.go`
- `internal/api/contract/channels.go`
- `internal/api/contract/contract.go`
- `internal/api/core/channels.go`
- `internal/api/core/conversions.go`
- `internal/api/httpapi/channels_integration_test.go`
- `internal/daemon/channels.go`
- `internal/daemon/daemon.go`
- `internal/extension/manager.go`
- `internal/extension/host_api_channels.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`
- `web/src/systems/daemon/`

## Errors / Corrections
- The first combined coverage run exposed a real signal bug: broker retry handling overwrote terminal delivery errors with `channels: delivery transport unavailable`; fixed by preserving terminal `error` events as the instance `last_error`.
- `make verify` initially failed on stale generated API artifacts and on daemon-health web fixtures/types that still modeled the old contract; fixed with `make codegen` plus targeted web test updates.
- `make verify` also surfaced a `staticcheck` append-loop warning in `internal/observe/channels.go`; fixed without behavior changes.

## Ready for Next Run
- Task implementation, verification, and memory updates are complete.
- Remaining closeout is limited to task tracker updates and the single local completion commit.

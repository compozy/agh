# Task Memory: task_25

## Objective Snapshot

- Implement notification cursor diagnostics and bridge subscription lifecycle semantics for `.compozy/tasks/orch-improvs/task_25.md`.
- Success requires cursor diagnostics in bridge subscription read models/API/CLI output, delivered/deferred/fail-closed notifier state coverage, lifecycle cleanup/replay/stale-cursor tests, generated contract co-ship if DTOs change, and final `make verify` PASS.

## Important Decisions

- Cursor diagnostics are read-only projections over `notification_cursors` plus active `bridge_task_subscriptions`; `internal/notifications` remains a cursor/progress boundary, not a task event bus or bridge target authority.
- Bridge subscription delete removes the active subscription and returns deterministic not-found errors afterward; stale cursor diagnostics remain preserved for operator/agent inspection and same-subscription replay continuity.
- The terminal notifier now distinguishes `defer` from `mismatch`: non-terminal task state defers without cursor mutation, while accepted-final status disagreement fails closed, records bounded `last_error`, and does not advance `last_sequence`.

## Learnings

- ADR-003 defines the notifier decisions as `deliver`, `defer`, and `mismatch`; `mismatch` fails closed by recording bounded cursor error and not advancing the cursor.
- Task 21 intentionally left full cursor diagnostic expansion to task 25 after establishing the canonical `/api/tasks/{id}/notifications/bridges` transport shape.
- Existing API/CLI baseline exposed only cursor identity (`consumer_id`, `stream_name`, `subject_id`); task 25 expands the same `cursor` object with `last_sequence`, `last_delivery_id`, `last_delivered_at`, `last_error`, and `updated_at`.
- Existing notifier baseline treated terminal status disagreement as an ordinary deferred event. That hid fail-closed mismatches until this task added explicit mismatch diagnostics.

## Files / Surfaces

- `internal/bridges`
- `internal/notifications`
- `internal/store/globaldb`
- `internal/api/contract`
- `internal/api/core`
- `internal/api/httpapi`
- `internal/api/udsapi`
- `internal/cli`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections

- Running `make lint` after the notifier change exposed `funlen` in `deliverSubscription` and an overlong mismatch diagnostic line. Fixed by extracting cursor/event record loading helpers and wrapping the diagnostic string.
- Running `make bun-typecheck` concurrently with `make lint` exposed a local Mage temp-output race (`mage_output_file.go` missing). Reran the gates sequentially; `make lint` and `make bun-typecheck` both passed.

## Completion Evidence

- Focused tests passed:
  - `go test ./internal/bridges -run TestTerminalTaskNotifierDeliverDue -count=1`
  - `go test ./internal/store/globaldb -run TestGlobalDBBridgeTaskSubscriptionStore -count=1`
  - `go test ./internal/api/core -run TestBaseHandlersTaskBridgeNotificationSubscriptionEndpoints -count=1`
  - `go test ./internal/cli -run TestTaskNotificationCommandsMapRequests -count=1`
  - `go test ./internal/api/contract ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1`
  - `go test ./internal/bridges ./internal/store/globaldb ./internal/notifications -count=1`
  - `go test ./internal/daemon -run '^$' -count=1`
- Contract/gates passed:
  - `make codegen`
  - `make codegen-check`
  - `make lint`
  - `make bun-typecheck`
  - `git diff --check`
- Final gate passed: `make verify` completed with Vitest 329 files / 2092 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8283 tests in 136.798s`, and `OK: all package boundaries respected`.

## Ready for Next Run

- Task 25 is complete.
- Next: execute `task_26` for the web generated-client/data-layer slice. The generated OpenAPI/TypeScript contracts now include bridge notification cursor diagnostics for downstream web consumption.

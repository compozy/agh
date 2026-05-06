---
status: completed
title: "Notification Cursor Diagnostics and Bridge Subscription Lifecycle"
type: backend
complexity: high
dependencies:
  - task_06
  - task_07
  - task_21
  - task_24
---

# Task 25: Notification Cursor Diagnostics and Bridge Subscription Lifecycle

## Overview
This task completes notification diagnostics and lifecycle semantics around bridge task subscriptions. It must expose cursor diagnostics, prove the terminal notifier has deliver/defer/fail-closed states, and ensure operators and agents can inspect stuck subscriptions.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose notification cursor diagnostics for bridge task subscriptions.
- MUST prove terminal notifier states: delivered, deferred without mismatch, and fail-closed with diagnostic error.
- MUST provide lifecycle operations and deterministic errors for subscription cleanup.
</requirements>

## Subtasks
- [x] Add cursor diagnostic read models and API/CLI output fields.
- [x] Add tests for delivered, deferred-no-mismatch, and fail-closed notifier states.
- [x] Add lifecycle edge tests for deleted subscriptions, replay, and stale cursor state.
- [x] Verify CLI/HTTP/UDS output parity for diagnostics.
- [x] Run codegen and verification gates when contracts change.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/bridges` - terminal notifier state machine.
- `internal/notifications` - cursor service.
- `internal/store/globaldb/global_db_notification_cursor.go` - cursor diagnostics.
- `internal/api/core` - diagnostic handlers.

### Dependent Files
- `internal/cli/task.go` - CLI diagnostics.
- `openapi/agh.json` - generated contract.
- `web/src/generated/agh-openapi.d.ts` - generated types.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursors are monotonic and replay-safe.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Makes cursor state observable for bridge extensions and operators.
- Agent manageability: Agents can diagnose and clean bridge subscription state through CLI/HTTP/UDS.
- Config lifecycle: No config keys expected.

### Web/Docs Impact
- `web/`: Diagnostics fields feed notification panels in task 27.
- `packages/site`: Notification cursor documentation planned in task 29.

## Deliverables
- Task implementation or documentation matching the requirements above.
- Focused unit tests with 80%+ coverage where code changes.
- Integration, contract, e2e, or docs-build tests proportional to the touched behavior.
- Updated workflow memory, QA evidence, generated artifacts, or site docs when applicable.

## Tests
- Unit tests:
  - [x] Validate the primary success path for this task.
  - [x] Validate malformed input, missing dependency, or authorization failure paths.
  - [x] Validate boundary conditions named by the related TechSpec and ADRs.
- Integration tests:
  - [x] Exercise the task through the owning service/transport boundary when applicable.
  - [x] Compare persisted state, generated contract output, or rendered docs/UI with runtime truth.
  - [x] Run race, codegen, site, web, or full verify gates listed by the touched surface.
- Test coverage target: >=80% for changed code paths; docs-only tasks require 100% checklist evidence against authored pages.
- All tests must pass.

## Completion Evidence
- Implemented `TaskBridgeNotificationCursorPayload` so bridge notification subscription create/list/show responses expose `consumer_id`, `stream_name`, `subject_id`, `last_sequence`, `last_delivery_id`, `last_delivered_at`, `last_error`, and `updated_at`.
- API shared core handlers now load persisted cursor diagnostics when the daemon bridge runtime exposes the cursor store; missing cursors return a zero-sequence diagnostic identity instead of inventing state.
- CLI `agh task notification list|show` now renders cursor sequence/error/timestamp diagnostics in human and structured output, with JSON/TOON inheriting the generated contract payload.
- `TerminalTaskNotifier` now proves delivered, deferred-without-mismatch, replay-noop, superseded-final, and fail-closed mismatch states. Mismatches record bounded `last_error` and do not advance `last_sequence`.
- GlobalDB lifecycle tests prove deleted subscriptions return deterministic not-found behavior while stale cursor diagnostics remain preserved for inspection and same-subscription replay continuity.
- Focused tests passed: `go test ./internal/bridges -run TestTerminalTaskNotifierDeliverDue -count=1`; `go test ./internal/store/globaldb -run TestGlobalDBBridgeTaskSubscriptionStore -count=1`; `go test ./internal/api/core -run TestBaseHandlersTaskBridgeNotificationSubscriptionEndpoints -count=1`; `go test ./internal/cli -run TestTaskNotificationCommandsMapRequests -count=1`; `go test ./internal/api/contract ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1`; `go test ./internal/bridges ./internal/store/globaldb ./internal/notifications -count=1`; `go test ./internal/daemon -run '^$' -count=1`.
- Contract and verification gates passed: `make codegen`; `make codegen-check`; `make lint`; `make bun-typecheck`; `git diff --check`; final `make verify` with Vitest 329 files / 2092 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8283 tests in 136.798s`, and package boundaries OK.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.

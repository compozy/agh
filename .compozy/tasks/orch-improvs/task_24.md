---
status: completed
title: "Latest Event Sequence and Cursor-Seeded Task SSE"
type: backend
complexity: high
dependencies:
  - task_05
  - task_23
---

# Task 24: Latest Event Sequence and Cursor-Seeded Task SSE

## Overview
This task adds latest-event sequence read models and cursor-seeded SSE behavior for task observers. It must implement read-then-stream semantics, honor `Last-Event-ID` precedence, and prevent notification gaps between initial snapshots and event streams.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, every ADR, and the dependency task files before starting.
- REFERENCE TECHSPEC for implementation details; do not duplicate architecture or code snippets here.
- FOCUS ON WHAT needs to be delivered, keep changes scoped, and avoid compatibility shims or fallback paths.
- TESTS REQUIRED: every production change must ship focused tests and run the listed verification gates.
</critical>

<requirements>
- MUST expose `latest_event_seq` in task/run context read models.
- MUST seed SSE streams from explicit cursor or `Last-Event-ID` with deterministic precedence.
- MUST prove read-then-stream has no missed terminal/review/notification events.
</requirements>

## Subtasks
- [x] Add latest event sequence projection to task context/read models.
- [x] Implement SSE seed parsing with explicit cursor and `Last-Event-ID` precedence.
- [x] Add read-then-stream snapshot plus replay semantics for task observers.
- [x] Cover stale, missing, malformed, and concurrent terminal-event cases with tests.

## Implementation Details
Required skill activation must match the touched surfaces: backend tasks use `agh-code-guidelines`, `golang-pro`, and `agh-test-conventions`; contract tasks also use `agh-contract-codegen-coship`; web tasks use the web instructions and frontend/design skills; docs tasks use `documentation-writer`, `copywriting` when public prose changes, and the site instructions; QA tasks use the QA skills named in the task. Use the TechSpec and ADRs for architecture; this task records scope and evidence boundaries.

### Relevant Files
- `internal/api/core/observe.go` - SSE observe handlers if task events share observe surface.
- `internal/events` - event sequence source.
- `internal/store/globaldb` - task event read models.
- `internal/task` - context bundle projection.

### Dependent Files
- `internal/api/httpapi/routes.go` - HTTP stream route.
- `internal/api/udsapi/routes.go` - UDS route parity if applicable.
- `web/src/generated/agh-openapi.d.ts` - generated stream contract types.

### Related ADRs
- [ADR-003: Durable Cursor Primitive](adrs/adr-003.md) - notification cursors are monotonic and replay-safe.
- [ADR-005: Denormalized Current Run Projection](adrs/adr-005.md) - current run read model is owned by GlobalDB transitions.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: Provides replay-safe stream semantics for bridge and web observers.
- Agent manageability: Agents can resume observation from deterministic event cursors.
- Config lifecycle: No config keys expected.

### Web/Docs Impact
- `web/`: Tasks 26 and 27 consume seeded task event streams.
- `packages/site`: SSE/cursor behavior documented in task 29.

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
- `latest_event_seq` is exposed from durable `task_events.event_seq` projections across task references, summaries, details, dashboard active runs, inbox items, and task context bundles.
- Task stream cursor parsing now gives a present `Last-Event-ID` header deterministic precedence over `?after_sequence=`, including `Last-Event-ID: 0`; malformed header values fail parsing.
- Read-then-stream replay is covered with tests that seed from a read snapshot, replay terminal/review/notification events after the seed, and deliver live terminal events after subscription without gaps.
- Context bundle recent events now preserve durable event sequence by reading bounded `ListTaskEventRecords(..., Descending: true)` and rendering them chronologically.
- Contract artifacts were regenerated with `make codegen` and checked with `make codegen-check`; generated web fixtures were updated to satisfy the strict `latest_event_seq` contract.
- Focused/package validation passed for `internal/task`, `internal/store/globaldb`, `internal/api/core`, `internal/situation`, and `internal/observe`.
- Bun and Go guardrails passed: `make bun-lint`, `make bun-typecheck`, `make bun-test`, `make lint`, `make codegen-check`.
- Final gate passed: `make verify` reported Go race `DONE 8279 tests in 135.206s` and `OK: all package boundaries respected`.

## Success Criteria
- All tests passing.
- Test coverage >=80% for changed code paths, or 100% documented evidence coverage for docs-only tasks.
- `make verify` passes before the task is marked complete.
- The task evidence is recorded in workflow memory or QA artifacts.

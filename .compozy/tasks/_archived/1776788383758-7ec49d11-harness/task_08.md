---
status: completed
title: Harness observability, event summaries, and integration hardening
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
  - task_05
  - task_07
---

# Task 08: Harness observability, event summaries, and integration hardening

## Overview

Finish the harness architecture by making its lifecycle visible and auditable through existing observe and storage surfaces, then harden the end-to-end integration paths. This task ensures startup section selection, augmentation, synthetic reentry, and detached completion all leave behind structured evidence and regression coverage instead of remaining opaque runtime behavior.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_02.md`, `task_03.md`, `task_05.md`, `task_07.md` before starting
- REFERENCE TECHSPEC sections "Workstream 6: Storage, Observability, and Verification", "Monitoring and Observability", and "Impact Analysis"
- FOCUS ON "WHAT" - make harness lifecycle visible through existing event-summary and integration surfaces; do not create a second observe subsystem
- MINIMIZE CODE - use `EventSummary` and existing observe/read-side seams wherever they already fit
- TESTS REQUIRED - event-summary emission, query visibility, and integration hardening across prompt, reentry, and detached flows are mandatory
- GREENFIELD: nenhum estado importante do harness pode ficar invisivel; se o runtime decidir algo, isso precisa aparecer em observabilidade
</critical>

<requirements>
- MUST emit structured observability for harness context resolution, startup section selection, augmentation, detached completion, and synthetic reentry
- MUST reuse `EventSummary` and current observer/globaldb seams instead of creating a separate observability store
- MUST keep read-side visibility available through current observe/query surfaces
- MUST add integration hardening coverage across the final harness flow, not just isolated unit tests
- SHOULD document schema or read-side implications clearly where new event types or summary types are introduced
</requirements>

## Subtasks
- [x] 8.1 Add harness lifecycle event-summary emission for the major runtime decisions and outcomes
- [x] 8.2 Extend observe/query read-side behavior so harness events are inspectable and stable
- [x] 8.3 Add storage and summary coverage for synthetic reentry and detached completion outcomes
- [x] 8.4 Harden integration tests across startup, augmentation, transcript, detached completion, and reentry
- [x] 8.5 Ensure verification gates for the full harness slice are explicit and repeatable

## Implementation Details

See TechSpec "Workstream 6: Storage, Observability, and Verification" plus the "Monitoring and Observability" section. This task is where the harness becomes operable: engineering should be able to answer what the runtime decided, what reentered, what stayed silent, and why.

### Relevant Files
- `internal/observe/observer.go` - current event-summary write path that harness lifecycle events must reuse
- `internal/store/globaldb/global_db_observe.go` - durable event-summary storage and list behavior
- `internal/observe/query.go` - read-side query entrypoint that later callers will use for harness inspection
- `internal/observe/tasks.go` - task-summary surfaces likely to intersect with detached harness work
- `internal/api/core/session_stream.go` - existing stream/read surfaces that may surface the hardened harness flow indirectly
- `internal/daemon/daemon_nightly_combined_integration_test.go` - good integration lane for the broader hardened runtime flow

### Dependent Files
- `internal/observe/observer_test.go` - direct coverage for new harness summary emission
- `internal/store/globaldb/global_db_extra_test.go` - summary storage edge cases and ordering behavior may need updates
- `internal/api/httpapi/stream_helpers_test.go` - stream-related summary behavior may need assertions if new event types surface
- `internal/api/udsapi/stream_helpers_test.go` - UDS summary visibility should remain aligned with HTTP
- `internal/daemon/daemon_integration_test.go` - final harness integration may need a consolidated runtime assertion path

### Related ADRs
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - Observability must cover both startup and turn-time seams
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - Detached completion and synthetic wake-up must be visible in the same observability model
- [ADR-004: Defer Coordinator-Grade Orchestration Contracts from the Harness Architecture Phase](adrs/adr-004.md) - Observability should support later orchestration work without prematurely hard-coding coordinator semantics

### External References
- `.resources/openclaw/src/agents/anthropic-payload-log.ts` - useful model for structured payload logging with digest/redaction discipline
- `.resources/openclaw/docs/automation/tasks.md` - practical reference for observable async task states and delivery semantics
- `.resources/hermes/gateway/status.py` - useful precedent for runtime status surfacing and operational assertions
- `.resources/hermes/hermes_cli/logs.py` - strong reference for follow/tail log ergonomics and session-scoped inspection
- `.resources/openfang/crates/openfang-kernel/src/event_bus.rs` - helpful model for observable internal event routing and history
- `.resources/openfang/docs/api-reference.md` - useful QA/read-side reference for the operator-facing shape of async/eventful runtime surfaces
- `.resources/claude-code/main.tsx` - startup wiring reference for where runtime addenda and event flow stay inspectable

## Deliverables
- Harness lifecycle event-summary emission across context resolution, section selection, augmentation, detached completion, and synthetic reentry
- Read-side visibility for the new harness signals through existing observe/query surfaces **(REQUIRED)**
- Hardened integration coverage for the final harness runtime flow **(REQUIRED)**
- Explicit verification gate expectations captured in task-local tests and deliverables **(REQUIRED)**
- Regression coverage for summary ordering, filtering, and parity across observer-backed read surfaces **(REQUIRED)**
- Unit and integration tests with >=80% coverage for modified observe/store paths **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Context resolution and startup section selection emit the expected harness-specific event-summary entries
  - [x] Augmenter warning and failure paths surface structured harness observability without corrupting dispatch
  - [x] Synthetic reentry and silent completion produce distinct summary types or messages that are queryable later
  - [x] Event-summary ordering, truncation, and query filtering remain stable after the new harness event types are introduced
  - [x] Observer and globaldb summary writers preserve session id, agent name, and timestamp semantics for harness events
- Integration tests:
  - [x] End-to-end detached completion plus synthetic reentry flow is visible through observer and query surfaces with the expected ordering
  - [x] HTTP and UDS transport parity surfaces expose equivalent harness-summary visibility after the runtime flow executes
  - [x] Combined startup, prompt, transcript, detached completion, and reentry scenarios remain green under the hardened integration lane
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The harness architecture is operationally observable through existing AGH surfaces
- Engineers can trace the major harness decisions and runtime outcomes without inspecting opaque process logs only

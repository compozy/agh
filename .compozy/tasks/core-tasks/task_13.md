---
status: pending
title: "Add observe projections, health queries, and task metrics"
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
  - task_12
---

# Task 13: Add observe projections, health queries, and task metrics

## Overview
Add the read-side observability needed to operate the new task domain confidently. This task should project task and run lifecycle information into health, query, and metrics surfaces so the daemon can report queue depth, stuck work, task ownership, and channel-aware operational state.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. The observe subsystem MUST project task and run lifecycle data into queryable health and summary views without becoming a second lifecycle authority.
2. Observability outputs MUST include queue depth, stuck runs, task totals, and channel/origin-aware operational signals described in the TechSpec.
3. Metrics and queries MUST reflect network-originated channel behavior and recovery outcomes introduced by earlier tasks.
</requirements>

## Subtasks
- [ ] 13.1 Extend observe-side projections to consume task, run, and audit events.
- [ ] 13.2 Add query and health surfaces for queue depth, stuck runs, task totals, and ownership/channel filters.
- [ ] 13.3 Add metrics emission for task counts, run counts, queue latency, duplicate ingress, and channel mismatch events.
- [ ] 13.4 Integrate recovery and cancellation signals so health views reflect orphan recovery and forced-stop outcomes.
- [ ] 13.5 Add tests covering channel-aware and origin-aware observability outputs.

## Implementation Details
Use the TechSpec sections "Monitoring and Observability", "Cold-Start Recovery", and "Cancellation Model". Follow the existing patterns in `internal/observe/observer.go`, `query.go`, `health.go`, and `reconcile.go`, plus the `global_db_observe.go` read-side storage helpers.

### Relevant Files
- `internal/observe/observer.go` — Existing observer/event ingestion surface.
- `internal/observe/query.go` — Existing query surface that task projections should extend.
- `internal/observe/health.go` — Existing health computation logic that task-aware health should join.
- `internal/observe/reconcile.go` — Existing read-side reconciliation patterns relevant to task projections.
- `internal/store/globaldb/global_db_observe.go` — Existing observe-facing storage helpers that may need task queries.
- `internal/network/audit.go` — Source of network audit signals that task metrics should account for.
- `internal/task/` — Source of manager events and lifecycle outputs consumed by observe.

### Dependent Files
- `internal/api/core/handlers.go` — May later consume task health/query outputs for user-facing read paths.
- `internal/daemon/boot.go` — Will need to wire task observers and metrics registration into daemon startup.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Requires observe to remain read-side only and not become a lifecycle authority.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Requires channel-aware metrics and health views.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](../adrs/adr-006.md) — Requires recovery and dedicated-session behavior to appear in observability outputs.

## Deliverables
- Observe projections and queries for tasks, runs, queue depth, and stuck work.
- Task and run metrics aligned with the TechSpec.
- Health coverage for cancellation, recovery, and channel-aware task behavior.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for task observability flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify task and run projections aggregate counts by status, scope, origin, and channel correctly.
  - [ ] Verify health logic flags stuck `claimed`, `starting`, or `running` task runs according to the configured rules.
  - [ ] Verify duplicate-ingress and channel-mismatch counters increment from the expected audit inputs.
- Integration tests:
  - [ ] Verify a full task lifecycle from queue to completion appears in observe queries and metrics with the expected channel and origin metadata.
  - [ ] Verify orphan-run recovery and forced-stop cancellation outcomes are reflected in task health views after daemon restart or shutdown scenarios.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Operators can inspect task health, queue state, and channel-aware metrics through the observe subsystem
- Observability remains read-side only while accurately reflecting task lifecycle, recovery, and ingress behavior

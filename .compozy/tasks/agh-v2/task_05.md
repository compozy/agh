---
status: completed
domain: Observability
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_02
  - task_04
---

# Task 05: Observe Package

## Overview

Implement the `internal/observe` package that records session events to the global database, tracks health metrics, and provides a query engine for cross-session observability. This package implements the `session.Notifier` interface to receive events from the session manager and writes summaries/stats to `agh.db`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `session.Notifier` interface (OnSessionCreated, OnSessionStopped, OnAgentEvent)
- MUST write event summaries to global `agh.db` event_summaries table
- MUST update token_stats aggregation in global DB on each turn
- MUST write permission decisions to permission_log in global DB
- MUST track health metrics: active sessions, active agents, daemon uptime, DB sizes
- MUST provide cross-session event query methods (by session, agent, type, time range)
- MUST support boot-time reconciliation (scan sessions dir, reconcile with agh.db)
</requirements>

## Subtasks
- [x] 5.1 Implement Notifier: OnSessionCreated registers session in global DB
- [x] 5.2 Implement Notifier: OnSessionStopped updates session state in global DB
- [x] 5.3 Implement Notifier: OnAgentEvent writes event summary + updates token stats
- [x] 5.4 Implement health metrics collection (active counts, uptime, DB sizes)
- [x] 5.5 Implement cross-session event query engine
- [x] 5.6 Implement boot-time reconciliation logic

## Implementation Details

Create the following files:
- `internal/observe/observer.go` — Notifier implementation, event recording to global DB
- `internal/observe/health.go` — Health metrics collection
- `internal/observe/query.go` — Cross-session query engine
- `internal/observe/reconcile.go` — Boot-time reconciliation logic

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — Write Path Ownership, Monitoring section, Failure Handling

### Old Project Reference
- `.old_project/internal/observability/runtime.go` — Event recording and metrics collection patterns
- `.old_project/internal/observability/transcript.go` — Session recording patterns
- `.old_project/internal/observability/types.go` — Event type definitions

### Related ADRs
- [ADR-006: Dual SQLite Storage](../adrs/adr-006.md) — Global DB owns summaries and stats
- [ADR-008: Direct Interfaces and Notifier Pattern](../adrs/adr-008.md) — Notifier implementation
- [ADR-009: Agent-First Observability](../adrs/adr-009.md) — CLI-queryable, structured output

## Deliverables
- `internal/observe/` package with Notifier implementation, health, queries, reconciliation
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with real SQLite **(REQUIRED)**

## Tests
- Unit tests:
  - [x] OnSessionCreated: registers session in global DB
  - [x] OnSessionStopped: updates session state to stopped
  - [x] OnAgentEvent: writes event summary to global DB
  - [x] OnAgentEvent: updates token_stats with nullable values
  - [x] Health metrics: returns correct active session/agent counts
  - [x] Cross-session query: filter by session ID
  - [x] Cross-session query: filter by event type
  - [x] Cross-session query: filter by time range
  - [x] Cross-session query: limit results
  - [x] Reconciliation: session dir not in DB gets indexed
  - [x] Reconciliation: DB entry with no dir gets marked orphaned
- Integration tests:
  - [x] Full flow: notify events → query cross-session → verify results
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Global DB correctly reflects session state after Notifier calls
- Cross-session queries return accurate filtered results
- Reconciliation correctly handles crash recovery scenarios

---
status: completed
title: Persist Heartbeat Snapshots, Revisions, Session Health, and Wake Audit
type: backend
complexity: critical
dependencies:
  - task_02
  - task_05
---

# Task 06: Persist Heartbeat Snapshots, Revisions, Session Health, and Wake Audit

## Overview

Add durable storage for Heartbeat policy snapshots, authoring revisions, metadata-only session health, and wake audit/state. This task creates the persistence foundation for managed `HEARTBEAT.md`, scheduler decisions, observability, and route parity without creating a parallel work queue.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, and ADR-007 through ADR-011 before changing persistence.
- REFERENCE TECHSPEC for migration v13 DDL, standalone `session_health`, wake audit/state, retention, and delete targets.
- FOCUS ON WHAT must be stored: policy snapshots, revisions, session health rows, wake state, wake events, and diagnostics.
- MINIMIZE CODE in notes; follow existing store and migration conventions.
- TESTS REQUIRED for fresh DB, reopen, stale rows, constraints, retention, and no queue semantics.
- NO WORKAROUNDS: do not model wake policy as `task_runs`, `agent_wakeup_requests`, `heartbeat_runs`, or any ownership queue.
</critical>

<requirements>
- MUST activate `agh-schema-migration`, `agh-code-guidelines`, and `golang-pro`.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST add the numbered global DB migration specified by `_techspec_heartbeat.md` after the Soul migration.
- MUST create durable `agent_heartbeat_snapshots`, `agent_heartbeat_revisions`, `session_health`, `agent_heartbeat_wake_state`, and `agent_heartbeat_wake_events` tables or exact equivalents from the spec.
- MUST store `heartbeat_digest`, config digest/provenance, status, prompt contribution metadata, active-hours resolution, diagnostics, and revision history.
- MUST store session health as metadata-only runtime state, separate from authored `HEARTBEAT.md`.
- MUST add retention/cascade behavior and closed wake reason fields as specified.
- MUST avoid any schema that can be interpreted as a task queue, lease owner, or alternate `ClaimNextRun`.
</requirements>

## Subtasks
- [x] 6.1 Add the Heartbeat migration and register it after Soul storage migrations.
- [x] 6.2 Add store methods for Heartbeat snapshots and authoring revisions.
- [x] 6.3 Add store methods for `session_health` upserts, reads, stale marking, and restart recovery inputs.
- [x] 6.4 Add store methods for wake state and wake event audit rows with retention.
- [x] 6.5 Add migration, store, constraint, retention, and no-queue tests.
- [x] 6.6 Verify storage does not touch task-run lease heartbeat, scheduler claim state, or network greet rows.

## Implementation Details

Use side tables for queryable runtime state and provenance. Arbitrary metadata JSON may carry supplemental details, but it must not be the authority for session health, wake eligibility, digest, run reason, or retention fields.

### Relevant Files
- `internal/store/globaldb/global_db.go` - migration registration and global DB boot.
- `internal/store/globaldb/` - Heartbeat snapshot, revision, session health, wake state, and wake event store methods.
- `internal/heartbeat/` - policy snapshot and diagnostic types from task_05.
- `internal/session/` - session health identifiers and state types consumed by task_07.
- `internal/task/lease_manager.go` - read only to ensure no `task_runs` ownership schema is reused.

### Dependent Files
- `internal/store/globaldb/*_test.go` - migration and store behavior coverage.
- `internal/heartbeat/*_test.go` - persisted policy shape tests if shared fixtures are needed.
- `internal/session/*_test.go` - health persistence tests once task_07 consumes the store.
- `.compozy/tasks/agent-soul/task_07.md` - implements session health behavior on this schema.
- `.compozy/tasks/agent-soul/task_08.md` - uses Heartbeat revisions for managed authoring.
- `.compozy/tasks/agent-soul/task_09.md` - uses wake state and wake event audit data.

### Related ADRs
- [ADR-008: Heartbeat Snapshots and Wake Audit](adrs/adr-008.md) - defines storage without a queue.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - defines session health persistence.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - requires effective config provenance.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: persisted state must support Host API reads, hook payloads, tools/resources, SDKs, and audit resources without coupling extensions to DB internals.
- Agent manageability: no route exposed here, but store methods must support later CLI/HTTP/UDS and Host API status/history surfaces.
- Config lifecycle: persist effective `[agents.heartbeat]` config digest/provenance for policy snapshots and wake decisions.

### Web/Docs Impact
- Web impact: generated types and frontend consumers are deferred to tasks 10 and 14.
- Docs impact: task_15 must document audit/status behavior and make clear that raw tables are internal.

## Deliverables
- Numbered global DB migration for Heartbeat snapshots, revisions, session health, wake state, and wake events.
- Store methods with deterministic constraints, indexes, retention, and stale/restart support.
- Tests proving schema behavior, no queue semantics, and no collision with Soul migration numbering.
- Completion evidence that task-run lease heartbeat and AGH Network greet storage remain unchanged.

## Tests
- Unit tests:
  - [x] Fresh DB migration creates every Heartbeat and session health table with expected indexes.
  - [x] Reopen preserves Heartbeat snapshots, revisions, session health, wake state, and wake events.
  - [x] Wake reason enum accepts only closed values from the TechSpec.
  - [x] Retention deletes or compacts only eligible wake audit rows.
  - [x] Session health rows can be marked stale without deleting authored policy snapshots.
  - [x] Schema has no queue ownership, claim token, or task-run lease columns for Heartbeat wake policy.
- Integration tests:
  - [x] Resolved Heartbeat policy from task_05 persists and reads back with the same digest and config provenance.
  - [x] Failed migration returns a wrapped error and does not mark migration success.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_heartbeat.md` - normative storage, migration, and audit requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_paperclip_heartbeat.md` - Paperclip storage contrast and mismatch.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes_heartbeat.md` - Hermes task-run lease heartbeat contrast.
- `.resources/paperclip/packages/db/src/schema/agent_wakeup_requests.ts:5-40` - queue pattern to explicitly avoid.
- `.resources/paperclip/packages/db/src/schema/heartbeat_runs.ts:6-82` - run table pattern to explicitly avoid as AGH authority.
- `.resources/hermes/hermes_cli/kanban_db.py:305-330` - durable task-run heartbeat contrast.
- `.resources/hermes/hermes_cli/kanban_db.py:1183-1300` - task-run heartbeat persistence contrast.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Heartbeat and session health persistence supports wake decisions and status surfaces without creating a new work queue.
- Storage remains separate from task-run lease heartbeat, scheduler claim authority, and network greet presence.

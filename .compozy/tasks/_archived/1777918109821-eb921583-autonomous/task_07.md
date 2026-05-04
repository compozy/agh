---
status: completed
title: Task Claim Lease Schema
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 07: Task Claim Lease Schema

## Overview
Extend the existing task run model so pending work can be claimed safely by agents without introducing a parallel queue. This task is schema and type foundation only: it adds durable lease metadata, coordination channel association, capability indexes, and read-model redaction that later services and CLI verbs will consume.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, ADR-009, ADR-010, ADR-011, and ADR-012 before changing task run state
- PRESERVE `task_runs` AS THE DURABLE SOURCE OF TRUTH - do not create a second queue table
- DO NOT ADD DUPLICATE OWNER FIELDS - reuse existing `session_id`, actor/origin, queued/claimed timestamps where they already exist
- NEVER EXPOSE RAW `claim_token` THROUGH LIST/GET READ MODELS - only synchronous claim responses may include it
- TESTS REQUIRED - schema, type conversion, redaction, capability indexes, and restart reads must be covered
- NO WORKAROUNDS - do not hide races with sleeps, broad retries, or best-effort string matching
</critical>

<requirements>
- MUST add claim lease fields to the canonical task run model: `claim_token`, `claim_token_hash`, `lease_until`, and `heartbeat_at` where appropriate.
- MUST add durable `coordination_channel_id` to workspace-scoped coordinated task runs and index it for channel-to-run correlation.
- MUST keep raw `claim_token` out of persisted public DTO/read responses; list/get surfaces expose only token hash or boolean lease state.
- MUST add exact-match capability side tables for required and preferred task-run capabilities.
- MUST add indexes needed for pending-claim scans, active lease recovery, capability filtering, task lookup, and session lookup.
- MUST keep schema greenfield and direct; no legacy migration compatibility paths are required.
- MUST co-emit or prepare task-run hook payload fields through the typed hook bridge added in task_03.
</requirements>

## Subtasks
- [x] 7.1 Extend `internal/task.Run` and validation helpers with lease fields, `coordination_channel_id`, and capability requirements.
- [x] 7.2 Update global DB schema for task run lease metadata, coordination channel metadata, capability side tables, and targeted indexes.
- [x] 7.3 Update store scan/get/list helpers to round-trip lease fields, coordination channel ID, and capability rows atomically.
- [x] 7.4 Update contract DTOs and generated OpenAPI/web types if run read models expose new lease state.
- [x] 7.5 Add redaction tests proving raw claim tokens are never returned through normal read APIs.
- [x] 7.6 Add schema and store tests for restart reads, capability joins, and existing task-run lifecycle regressions.

## Implementation Details
Use the existing task store and `task_runs` table as the only durable work queue. The schema should make `ClaimNextRun` cheap for task_08, but task_07 should not implement claim-next behavior yet.

`coordination_channel_id` is correlation metadata for coordinator/worker communication. It must not become a second ownership path and must not authorize task state transitions.

Capabilities must live in side tables rather than JSON blobs so task_08 can use exact matching without parsing ad hoc payloads inside the claim transaction.

### Relevant Files
- `internal/task/types.go` - canonical `Run` fields and capability metadata.
- `internal/task/validate.go` - validation for lease/capability inputs.
- `internal/task/events.go` - task-domain event payloads affected by new run state.
- `internal/store/globaldb/global_db.go` - `task_runs` schema and table creation.
- `internal/store/globaldb/global_db_task*.go` - task run persistence, scans, and test helpers.
- `internal/api/contract/tasks.go` - run DTO redaction and generated contract source.
- `internal/hooks/payloads.go` - typed hook payload additions from task_03.
- `.resources/paperclip/doc/plans/2026-02-20-issue-run-orchestration-plan.md` - reference for issue-run orchestration and lease semantics.
- `.resources/hermes/hermes_state.py` - reference for durable orchestration state that survives process restarts.
- `.resources/multica/packages/core/issues/store.ts` - reference for indexed issue/run state and capability-like filtering.

### Dependent Files
- `internal/task/manager.go` - task_08 implements claim-next and lease fencing on top of this schema.
- `internal/api/udsapi/routes.go` - task_09 exposes agent task lease APIs.
- `internal/cli/task.go` - task_09 exposes agent lease CLI verbs.
- `web/src/generated/agh-openapi.d.ts` - regenerated if public contract changes.

### Related ADRs
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - authoritative lease schema and invariants.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - task-run hook payload requirements.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual and autonomous runs share one model.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - generated contract discipline.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - task-run channel association.

## Deliverables
- Updated task run schema and Go model with lease metadata.
- Durable `coordination_channel_id` field and index for coordinated task runs.
- Capability side tables and indexes for exact matching.
- Redacted read models that never leak raw claim tokens.
- Unit tests with 80%+ coverage for touched task/store mapping code **(REQUIRED)**.
- Store integration tests proving SQLite schema, indexes, and restart reads **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] `Run` validation accepts valid lease metadata and rejects malformed token hashes, past invalid lease windows, and invalid capability names.
  - [x] Store mappers preserve `lease_until`, `heartbeat_at`, required capabilities, and preferred capabilities without dropping existing actor/origin fields.
  - [x] Store mappers preserve `coordination_channel_id` and can query runs by coordination channel.
  - [x] Public DTO conversion exposes `claim_token_hash` or lease state but never raw `claim_token`.
  - [x] Empty capability sets and multiple capability rows round-trip deterministically.
  - [x] Existing create/enqueue/get/list task-run tests still pass without compatibility branches.
- Integration tests:
  - [x] SQLite schema creates `task_run_required_capabilities` and `task_run_preferred_capabilities` with indexes used by pending-run queries.
  - [x] A task run persisted with lease fields can be read after reopening the database.
  - [x] A task run persisted with `coordination_channel_id` can be read after reopening the database and does not expose raw claim tokens through channel-correlated read models.
  - [x] Listing runs for a session/task preserves active lease metadata without exposing raw tokens.
  - [x] Contract generation, web typecheck, and web tests pass if public DTOs change.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- The database has one durable task-run queue with explicit lease metadata.
- Later claim logic can be implemented with transactional SQLite updates and exact capability matching.

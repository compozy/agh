---
status: completed
title: Persistent task fields and lifecycle reconciliation
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 02: Persistent task fields and lifecycle reconciliation

## Overview

Persist the expanded task semantics and teach the task manager how they change lifecycle behavior. This task is the write-model backbone for the feature: it must make draft publication, attempt exhaustion, triage state, and approval-aware transitions real in durable storage and reconciliation logic.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_01.md` before changing lifecycle code
- REFERENCE TECHSPEC sections "Data Models", "Persistent storage changes", and "Development Sequencing"
- FOCUS ON "WHAT" — persist the new fields and define their lifecycle impact, not transport concerns
- MINIMIZE CODE — extend the existing task registry and manager instead of inventing a parallel write store
- TESTS REQUIRED — lifecycle transitions, durable round-trips, and reconciliation rules all need coverage
- GREENFIELD: prefer one clear lifecycle model for draft/approval/attempt exhaustion; do not add compat shims for the old ready-only assumptions
</critical>

<requirements>
- MUST persist `priority`, `max_attempts`, `approval_policy`, and `approval_state` on the durable task record
- MUST add durable triage state storage scoped by task and actor for read/unread, archived, dismissed, and last-seen activity
- MUST update manager lifecycle logic so draft tasks remain non-runnable until publication and published tasks reconcile into `ready` or `blocked`
- MUST enforce task-level attempt exhaustion and approval gating as part of lifecycle reconciliation
- MUST preserve existing run history as authoritative while treating task-level policy as configuration
- SHOULD keep boot and reconciliation behavior deterministic so later dashboard and inbox reads do not depend on ad hoc client logic
</requirements>

## Subtasks
- [x] 2.1 Extend durable task storage and schema handling for the new task fields
- [x] 2.2 Add durable triage-state persistence keyed by task and actor
- [x] 2.3 Update manager create/update/reconcile flows for draft and approval-aware transitions
- [x] 2.4 Enforce task-level attempt exhaustion and retry policy in lifecycle handling
- [x] 2.5 Add storage and manager tests for durable round-trips and transition matrices

## Implementation Details

See TechSpec sections "Persistent storage changes", "Data Models", and ADR-002/ADR-004. Keep the write model authoritative: dashboard, inbox, and UI flows should consume durable semantics rather than infer them from loosely related run or metadata fields.

### Relevant Files
- `internal/task/manager.go` — task creation, updates, reconciliation, and run-lifecycle logic that must honor draft, approval, and attempt policy
- `internal/task/manager_test.go` — core lifecycle test matrix for transitions, retries, and dependency updates
- `internal/task/manager_integration_test.go` — durable manager behavior against the real store
- `internal/store/globaldb/global_db_task.go` — task persistence and filtering logic for the primary task record
- `internal/store/globaldb/global_db_task_aux.go` — auxiliary task storage and query helpers that can host triage-state support
- `internal/store/globaldb/global_db.go` — schema/bootstrap surface when new durable structures are added

### Dependent Files
- `internal/task/types.go` — task_01 defines the semantic fields that this task persists and reconciles
- `internal/task/interfaces.go` — store capabilities may need expansion for triage and publication support
- `internal/observe/tasks.go` — dashboard and inbox projections will consume the durable state added here
- `internal/api/core/tasks.go` — handlers in task_08 will depend on these lifecycle operations and point reads

### Related ADRs
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — Requires durable first-class task semantics and explicit draft publication
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Depends on durable triage and aggregate-ready write-side state

## Deliverables
- Durable storage support for the new task fields and triage state
- Manager lifecycle logic for draft publication, approval gating, and attempt exhaustion
- Schema and storage tests for task and triage round-trips **(REQUIRED)**
- Manager unit and integration coverage for transition matrices **(REQUIRED)**
- No client-side-only draft or inbox semantics remaining in the design surface

## Tests
- Unit tests:
  - [x] Draft tasks stay non-runnable until explicit publication
  - [x] Publishing a task yields `ready` or `blocked` based on current dependencies
  - [x] Approval-gated tasks cannot progress into runnable execution without the expected approval state
  - [x] Attempt exhaustion updates task/run state consistently after the configured limit is reached
- Integration tests:
  - [x] GlobalDB round-trips persist the new task fields and triage state without loss
  - [x] Manager integration tests verify draft creation, publication, retries, and approval-aware transitions against the real store
  - [x] Actor-scoped triage mutations survive reload and remain isolated per actor reference
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified manager and store files
- Draft, approval, attempts, and triage semantics are durable and manager-owned
- Later API/read-model tasks can expose these capabilities without inventing new backend state

---
status: completed
title: First-class task semantics and validation
type: backend
complexity: high
dependencies: []
---

# Task 01: First-class task semantics and validation

## Overview

Promote the Paper-critical task semantics into the core task domain so later storage, APIs, and UI flows are built on real capabilities instead of metadata conventions. This task defines the canonical types and validation rules for `priority`, `draft`, `max_attempts`, `approval_policy`, and `approval_state`.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the `analysis/*.md` notes before changing the task domain (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "Core Interfaces", and "Technical Considerations"
- FOCUS ON "WHAT" — define durable semantics and invariants, not transport-specific behavior
- MINIMIZE CODE — keep the first change centered on task-domain types, validation, and explicit defaults
- TESTS REQUIRED — every new enum, field, and validation rule must have targeted coverage
- GREENFIELD: nao esconder `priority`, `draft`, ou approval semantics em `metadata`; virar campo de primeira classe e requisito de dominio
</critical>

<requirements>
- MUST add first-class task-domain support for `priority`, `max_attempts`, `approval_policy`, and `approval_state`
- MUST add an explicit non-runnable draft status and validation rules that distinguish draft tasks from runnable lifecycle states
- MUST define validation rules for approval-state combinations so invalid policy/state pairs are rejected before persistence
- MUST keep `metadata` available for extensibility while preventing it from remaining the primary home for Paper-critical semantics
- SHOULD keep defaults explicit so later create/update flows can derive behavior without hidden implicit values
</requirements>

## Subtasks
- [x] 1.1 Add the new task-domain enums, fields, and supporting read-model types
- [x] 1.2 Define draft-specific and approval-specific validation invariants
- [x] 1.3 Normalize default values for newly introduced task semantics
- [x] 1.4 Align task-domain interfaces with the expanded type surface
- [x] 1.5 Add focused validation and interface-compatibility tests

## Implementation Details

See TechSpec sections "Data Models", "Core Interfaces", and ADR-002. Keep this task constrained to domain semantics and validation so persistence, handlers, and UI tasks can depend on a stable type system before behavior is layered on.

### Relevant Files
- `internal/task/types.go` — canonical home for task fields, statuses, query/read-model shapes, and supporting enums
- `internal/task/validate.go` — current validation entrypoint that must enforce the new semantic rules
- `internal/task/validate_test.go` — existing validation matrix that should expand with new field/status cases
- `internal/task/interfaces.go` — service/store interfaces that must remain aligned with the expanded task shape
- `internal/task/errors.go` — likely place to preserve clear validation and domain error boundaries

### Dependent Files
- `internal/task/manager.go` — lifecycle logic will consume the new semantics in task_02 and task_03
- `internal/store/globaldb/global_db_task.go` — persistence layer must store the new fields in task_02
- `internal/api/contract/tasks.go` — transport payloads must expose the new fields in task_07
- `internal/observe/tasks.go` — read-side grouping will need the new semantics in task_05 and task_06

### Related ADRs
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — Defines `priority`, durable draft semantics, and task-level attempt policy as first-class domain capabilities

## Deliverables
- First-class task-domain fields and enums for Paper-critical semantics
- Validation rules for draft, approval, and attempt-policy behavior
- Interface alignment across task-domain service/store boundaries **(REQUIRED)**
- Unit tests with >=80% coverage for new validation paths **(REQUIRED)**
- Integration tests that prove the expanded semantics are consumable by the task manager surface **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Valid `priority` values are accepted and invalid values return descriptive validation errors
  - [x] Draft tasks validate as non-runnable while runnable statuses reject draft-only combinations
  - [x] `max_attempts` rejects zero, negative, and unsupported boundary values
  - [x] Approval policy/state combinations reject impossible or ambiguous pairings
- Integration tests:
  - [x] The task manager interfaces compile and execute against the expanded task types without adapter drift
  - [x] In-memory or manager-level creation paths surface the new validation failures before persistence is attempted
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified task-domain files
- The task domain has explicit support for draft, priority, attempts, and approval semantics
- Later persistence and API tasks can build on stable domain invariants instead of metadata conventions

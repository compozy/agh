---
status: completed
title: "Bootstrap the `internal/task` domain"
type: backend
complexity: high
dependencies: []
---

# Task 01: Bootstrap the `internal/task` domain

## Overview
Create the new `internal/task` package as the canonical home for task coordination concepts before persistence or transport work begins. This task establishes the shared language, invariants, and package boundaries that every downstream task will depend on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. The new `internal/task` package MUST define the core domain types for `Task`, `TaskRun`, actor identity, ownership, origin, scope, lifecycle enums, and bounded graph limits from the TechSpec.
2. The package MUST define the interfaces consumed by the task domain, including store-facing contracts and the injected session bridge, without importing `internal/session` directly.
3. Package-level validation helpers MUST encode immutable fields, payload size limits, scope rules, and shared sentinel errors so downstream packages do not reimplement them.
</requirements>

## Subtasks
- [x] 1.1 Create the `internal/task` package and its initial file layout for types, interfaces, validation, and errors.
- [x] 1.2 Define the core domain structs and enums used throughout the TechSpec and ADRs.
- [x] 1.3 Define actor, origin, and ownership models with server-derived semantics and optional ownership.
- [x] 1.4 Define shared limit constants and validation helpers for sizes, tree depth, dependency count, and direct child count.
- [x] 1.5 Define the store and session bridge interfaces that downstream implementations must satisfy.

## Implementation Details
Use the TechSpec sections "Core Interfaces", "Actor and Identity Model", "Authorization Contract", "Data Models", and "Guardrails and Limits" as the source of truth. Keep the package boundary strict: `internal/task` defines its own interfaces and errors, while concrete implementations remain in `globaldb`, `daemon`, and transport packages.

### Relevant Files
- `internal/task/` — New package to create for domain types, validation, interfaces, and errors.
- `internal/session/interfaces.go` — Reference existing session-facing interface style when defining the injected bridge seam.
- `internal/daemon/boundary.go` — Reference composition-boundary patterns that the task package must fit into.
- `.compozy/tasks/core-tasks/_techspec.md` — Source of truth for domain boundaries, limits, and invariants.

### Dependent Files
- `internal/store/globaldb/` — Will implement the task store interfaces defined here.
- `internal/api/core/interfaces.go` — Will depend on the manager/service interfaces introduced here.
- `internal/daemon/boot.go` — Will wire concrete implementations into the new domain seam.

### Related ADRs
- [ADR-001: Separate Task Coordination Records from TaskRun Execution Records](../adrs/adr-001.md) — Establishes the split between coordination and execution entities.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs actor/origin/owner semantics.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](../adrs/adr-006.md) — Governs the bridge seam that must be defined here.

## Deliverables
- New `internal/task` package with domain structs, enums, validation helpers, interfaces, and sentinel errors.
- Shared limit and mutability constants aligned with the TechSpec.
- Compile-time interface assertions where appropriate.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for package boundary and interface assumptions **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Validate `global` vs `workspace` scope rules reject invalid `workspace_id` combinations.
  - [x] Validate immutable field helpers reject updates to `created_by`, `origin`, `scope`, `workspace_id`, and `parent_task_id`.
  - [x] Validate payload-size guardrails reject metadata over 16 KB and payload/result values over 64 KB.
  - [x] Validate graph-limit helpers reject depth over 8, dependencies over 32, and direct children over 64.
- Integration tests:
  - [x] Verify the package composes against a fake store and fake session bridge without importing `internal/session`.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/task` exists as the canonical domain package for task concepts
- Downstream tasks can depend on stable task types and interfaces without redefining invariants

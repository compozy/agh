---
status: completed
title: Workspace domain types, errors, and store/resolver interfaces
type: backend
complexity: low
dependencies: []
---

# Task 01: Workspace domain types, errors, and store/resolver interfaces

## Overview

Introduce the `internal/workspace/` package skeleton with persisted and resolved models, sentinel errors, and the `WorkspaceStore` / `WorkspaceResolver` contracts. This task establishes types and interfaces only so `store/` and the Resolver can compile against a stable API without implementing behavior yet.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from TechSpec)
- REFERENCE TECHSPEC for implementation details — do not duplicate interface bodies here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST add `internal/workspace/` with `Workspace`, `ResolvedWorkspace`, `SkillPath`, and package-level errors per TechSpec "Data Models" and "Error types"
- MUST define `WorkspaceStore` with CRUD and lookup methods per TechSpec "WorkspaceStore"
- MUST define `WorkspaceResolver` with `Resolve` and `ResolveOrRegister` per TechSpec "WorkspaceResolver"
- MUST keep `store/` free of workspace logic in this task — interfaces live in `workspace/` only
- SHOULD split files per TechSpec / ADR-004 (`workspace.go`, `store.go`) without implementing Resolver behavior
</requirements>

## Subtasks
- [x] 1.1 Create package layout and exported types for persisted `Workspace` and computed `ResolvedWorkspace`
- [x] 1.2 Add sentinel errors for not-found, missing root, name/path conflicts, and agent availability
- [x] 1.3 Define `WorkspaceStore` interface matching GlobalDB responsibilities in later tasks
- [x] 1.4 Define `WorkspaceResolver` interface for session/daemon injection
- [x] 1.5 Add compile-time checks or minimal constructors only if needed for exported API consistency

## Implementation Details

See TechSpec sections "Core Interfaces", "Data Models", and ADR-004 file layout. No SQLite or filesystem scanning in this task.

### Relevant Files
- `internal/store/global_db.go` — Future implementer of `WorkspaceStore` (reference for method names only)
- `internal/session/manager.go` — Today uses string workspace paths; will consume `WorkspaceResolver` in task_04

### Dependent Files
- `internal/store/schema.go` — Will add `workspaces` DDL in task_02
- `internal/workspace/resolver.go` — Implemented in task_03

### Related ADRs
- [ADR-004: New internal/workspace/ Package](adrs/adr-004.md) — Package boundary and file layout

## Deliverables
- New Go files under `internal/workspace/` with types, errors, and interfaces
- Table-driven unit tests for error identity (`errors.Is`) where exported errors are compared **(REQUIRED)**
- No behavioral Resolver or DB code in this task

## Tests
- Unit tests:
  - [x] `ErrWorkspaceNotFound` is distinct and matches `errors.Is` from API returning it
  - [x] Zero-value structs and JSON tags (if any wire types) remain stable for later tasks
- Integration tests:
  - [x] Not required for pure types; N/A for this task
- Test coverage target: >=80% for new package surface (or justify with minimal code + error tests)
  - Declarations-only package: `go test ./internal/workspace -cover` reports `coverage: [no statements]`; exported error/API tests provide the intended protection for this task's minimal surface
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/workspace/` after types land
- `make verify` passes when combined with dependent tasks’ completion (this task alone may be thin if only types)

---
status: completed
title: Observe and memory workspace ID references
type: backend
complexity: medium
dependencies:
  - task_04
---

# Task 09: Observe and memory workspace ID references

## Overview

Update `internal/observe` and `internal/memory` to use `WorkspaceID` for session reconciliation, metrics, and dream/memory paths. When a filesystem path is required, resolve it via `workspace.Resolver` rather than storing ambiguous path strings alone.

<critical>
- READ `_techspec.md` observe/memory impact rows
- TESTS REQUIRED
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST replace bare workspace path fields with `WorkspaceID` where sessions are indexed or reconciled
- MUST update dream/memory flows to obtain root path through Resolver lookup by ID
- MUST keep structured logging fields consistent (`workspace_id` in events per TechSpec)
- MUST update unit tests in observe/memory packages for new field names
</requirements>

## Subtasks
- [x] 9.1 Trace workspace string usage in `internal/observe` and update types/queries
- [x] 9.2 Update memory package to accept Resolver or resolved paths from daemon/session
- [x] 9.3 Fix compilation in all call sites from task_04 through task_06
- [x] 9.4 Extend tests for reconciliation and memory path resolution

## Implementation Details

See TechSpec Impact Analysis for `observe/` and `memory/`. Search codebase for `Workspace` field access after session struct change.

### Relevant Files
- `internal/observe/` — Session reconciliation, metrics
- `internal/memory/` — Dream and memory consolidation

### Dependent Files
- `internal/daemon/` — Already wired with Resolver (task_06)
- `internal/session/` — Emits workspace id on events

## Deliverables
- Observe/memory use workspace IDs; paths resolved via Resolver when needed
- Updated tests in affected packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Reconcile session with workspace id matches global DB row
  - [x] Missing workspace id yields defined error or skip behavior per design
  - [x] Memory/dream code resolves filesystem path only through Resolver
- Integration tests:
  - [x] If package has `integration` tests, update fixtures
- Test coverage target: >=80% per touched package (`internal/observe`, `internal/memory`)
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified packages
- `make verify` passes

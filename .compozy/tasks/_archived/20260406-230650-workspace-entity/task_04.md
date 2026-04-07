---
status: completed
title: Session Manager workspace ID and Resolver injection
type: backend
complexity: high
dependencies:
  - task_03
---

# Task 04: Session Manager workspace ID and Resolver injection

## Overview

Refactor `session.Manager` to depend on `workspace.WorkspaceResolver`, replace `resolveWorkspace()` path-only resolution with `ResolvedWorkspace`, and persist `WorkspaceID` on sessions and session metadata. This aligns session creation and resume with registered workspaces and removes `os.Getwd()` as the daemon-side default for empty workspace input.

<critical>
- READ `_techspec.md` session integration and ADR-001
- REFERENCE TECHSPEC for `CreateOpts` / resume behavior
- TESTS REQUIRED
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST add `WithWorkspaceResolver` (or equivalent) option to `session.Manager` and require Resolver for Create/Resume
- MUST change `Session` struct and session meta to carry `WorkspaceID` (not a free-form path string)
- MUST use `ResolvedWorkspace` for config, merged agents, and startup prompt assembly paths
- MUST remove `resolveWorkspace()` fallback that calls `os.Getwd()` for empty workspace — callers must pass explicit path or registered id/name (per TechSpec CLI/API)
- MUST update `CreateOpts` / resume inputs to accept workspace name, id, or path fields per TechSpec "Session creation change"
- MUST update all session tests (`manager_test.go`, `additional_test.go`, `query_test.go`, helpers) to use Resolver or test doubles
</requirements>

## Subtasks
- [x] 4.1 Add Resolver field and option; thread through `Create` and `Resume`
- [x] 4.2 Replace path string workspace with `WorkspaceID` in `Session` and persistence
- [x] 4.3 Wire `loadConfig` / `loadAgent` / assembler to use resolved root and merged definitions from Resolver output
- [x] 4.4 Delete or narrow `resolveWorkspace` helper; update error messages
- [x] 4.5 Fix tests and fakes across `internal/session/` and dependent test packages

## Implementation Details

See TechSpec "Data flow for session creation" and "Session schema change". `internal/daemon/daemon_test.go` and HTTP/UDS tests that assert `Workspace` field will need updates in later tasks or coordinated in this task if types change.

### Relevant Files
- `internal/session/manager.go` — `Create`, `Resume`, `resolveWorkspace`
- `internal/session/session.go` — Session fields
- `internal/store/` — Session meta read/write paths for workspace id
- `internal/session/manager_test.go` — Broad test coverage

### Dependent Files
- `internal/httpapi/` — Session create payload (task_10)
- `internal/acp/` — `StartOpts` extension (task_08)
- `internal/daemon/daemon_test.go` — Dream tests using `Session.Workspace`

### Related ADRs
- [ADR-001: Resolver with Persistent Backing](adrs/adr-001.md) — Session uses Resolver snapshots

## Deliverables
- Session lifecycle uses `WorkspaceID` and `ResolvedWorkspace`
- Updated unit tests across `internal/session/` **(REQUIRED)**
- No `Getwd()` workspace fallback in Manager paths

## Tests
- Unit tests:
  - [x] Create with registered workspace id resolves config and starts session
  - [x] Create with path triggers `ResolveOrRegister` and persists `WorkspaceID`
  - [x] Resume loads meta `WorkspaceID`, resolves workspace, fails with clear error if workspace row missing
  - [x] Empty workspace input returns validation error (not silent Getwd)
- Integration tests:
  - [x] Tagged tests if present: session create+resume with temp workspace dir
- Test coverage target: >=80% for `internal/session` (package threshold per CLAUDE.md)
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/session`
- `make verify` passes

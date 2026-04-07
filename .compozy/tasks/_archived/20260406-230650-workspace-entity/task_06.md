---
status: completed
title: Daemon wiring and dream consolidation
type: backend
complexity: medium
dependencies:
  - task_03
  - task_04
  - task_05
---

# Task 06: Daemon wiring and dream consolidation

## Overview

Wire `workspace.Resolver` and `GlobalDB` at the composition root in `internal/daemon`, pass Resolver into `session.Manager`, and update dream consolidation / recent-workspace logic to key off `WorkspaceID` instead of raw path strings.

<critical>
- READ `_techspec.md` "daemon/" impact and dream flow
- TESTS REQUIRED — update `daemon_test.go` fixtures
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST construct Resolver with correct dependencies (store, logger, config loader) in daemon boot
- MUST pass Resolver into session manager via options established in task_04
- MUST update dream spawner / consolidation to use `WorkspaceID` and resolve paths through Resolver when filesystem paths are needed
- MUST update `internal/daemon/daemon_test.go` and related tests that assert `Session.Workspace` string paths
- MUST preserve shutdown/lifecycle ordering — no leaked goroutines when daemon stops
</requirements>

## Subtasks
- [x] 6.1 Instantiate `GlobalDB` workspace APIs and Resolver in boot path
- [x] 6.2 Plumb Resolver into session manager and any skills/registry constructors needing it
- [x] 6.3 Refactor dream “recent workspaces” selection to use workspace IDs
- [x] 6.4 Update daemon tests for new session fields and workspace behavior
- [x] 6.5 Run `make verify` and fix regressions in `internal/daemon`

## Implementation Details

See TechSpec "daemon/" row in Impact Analysis and Development Sequencing step 7. Follow existing daemon patterns for dependency injection.

### Relevant Files
- `internal/daemon/` — Composition root (main daemon setup file(s))
- `internal/daemon/daemon_test.go` — Dream and session tests

### Dependent Files
- `internal/httpapi/` — Receives Resolver or session manager from daemon (task_10)
- `internal/cli/` — Talks to running daemon (task_12)

### Related ADRs
- [ADR-001: Resolver with Persistent Backing](adrs/adr-001.md)

## Deliverables
- Resolver wired at boot; dreams use workspace IDs
- Updated `daemon_test.go` coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Daemon boot with temp AGH home creates Resolver and session manager without error
  - [x] Dream consolidation test `TestDreamSpawnerDerivesRecentWorkspacesFromSessions` updated: expectations use workspace IDs
- Integration tests:
  - [x] Existing integration tests in daemon package still pass
- Test coverage target: >=80% for `internal/daemon` (per package rules)
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/daemon`
- `make verify` passes

---
status: completed
title: CLI workspace and session commands
type: backend
complexity: medium
dependencies:
  - task_06
  - task_11
---

# Task 12: CLI workspace and session commands

## Overview

Implement `agh workspace` subcommands (add, list, info, edit, remove) and extend `agh session` per TechSpec: `--workspace`, `--cwd`, list filtering, and default CWD auto-register behavior via daemon/UDS. Update CLI tests and help text.

<critical>
- READ `_techspec.md` "CLI Commands"
- TESTS REQUIRED — `session_test.go`, new workspace command tests
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST implement all workspace subcommands with flags per TechSpec
- MUST route operations through daemon UDS client (consistent with other CLI commands)
- MUST implement session flags: `--workspace`, `--cwd`, and no-flag behavior using caller CWD where specified
- MUST support `--force-refresh` or equivalent if exposed for Resolver cache bust (coordinate with task_03)
- MUST add table-driven tests for flag parsing and client request payloads
</requirements>

## Subtasks
- [x] 12.1 Add `workspace` command tree in `internal/cli/` and register in `root.go`
- [x] 12.2 Extend `session` command to send new workspace fields over UDS
- [x] 12.3 Update `internal/cli/client.go` for new request/response types
- [x] 12.4 Add `workspace_test.go` / extend `session_test.go` for coverage
- [x] 12.5 Run `make verify`

## Implementation Details

See TechSpec CLI section. Follow Cobra patterns in `internal/cli/session.go` and error formatting in `format.go`.

### Relevant Files
- `internal/cli/root.go` — Command registration
- `internal/cli/session.go` — Session subcommands
- `internal/cli/client.go` — UDS client
- `internal/cli/session_test.go` — Existing CLI tests

### Dependent Files
- `internal/udsapi/` — RPC surface (task_11)

## Deliverables
- User-facing `agh workspace` and updated `agh session` UX
- CLI unit tests **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `agh workspace add` builds request with root dir, optional name and add-dirs
  - [x] `agh session new --workspace ws_abc` sends workspace id to daemon
  - [x] `agh session new --cwd /tmp/proj` sends path for resolve/register
  - [x] Invalid flag combinations produce non-zero exit and stderr message
- Integration tests:
  - [x] `cli_integration_test.go` updated if it spins daemon
- Test coverage target: >=80% for `internal/cli` (package goal)
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/cli`
- `make verify` passes

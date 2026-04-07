---
status: completed
title: UDS API workspace mirror
type: backend
complexity: medium
dependencies:
  - task_10
---

# Task 11: UDS API workspace mirror

## Overview

Mirror the HTTP workspace and updated session commands over the UDS JSON-RPC (or message) API so CLI and local tools have feature parity with the web gateway. Follow existing `udsapi` patterns established for sessions.

<critical>
- READ `_techspec.md` udsapi impact
- KEEP payloads consistent with task_10
- TESTS REQUIRED — `udsapi/handlers_test.go`, `helpers_test.go`
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST expose workspace CRUD, resolve, and session create/list changes matching HTTP semantics
- MUST reuse the same validation and error mapping as HTTP layer where possible (shared functions in a small internal helper if DRY)
- MUST update `internal/udsapi/handlers_test.go` assertions that currently expect `Workspace: "/workspace"`
- MUST update CLI client types in `internal/cli/client.go` if they deserialize session/workspace fields
</requirements>

## Subtasks
- [x] 11.1 Add UDS methods or operations for each HTTP route (follow existing naming)
- [x] 11.2 Thread Resolver/deps into UDS handler construction parallel to HTTP
- [x] 11.3 Update tests mirroring `httpapi` cases for session and workspace
- [x] 11.4 Verify error codes map consistently for CLI consumers

## Implementation Details

See TechSpec "internal/udsapi/" row. Compare `internal/udsapi/` structure to `internal/httpapi/` for the session create path.

### Relevant Files
- `internal/udsapi/` — Handlers and router
- `internal/udsapi/handlers_test.go` — Behavioral tests
- `internal/udsapi/helpers_test.go` — Fixtures

### Dependent Files
- `internal/cli/client.go` — RPC client (task_12)

## Deliverables
- UDS parity with HTTP workspace + session contract
- Updated UDS tests **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Workspace register/list/get/update/delete via UDS matches HTTP outcomes for same inputs
  - [x] Session create with workspace id and with workspace_path both succeed
  - [x] Session list filter by workspace id works
- Integration tests:
  - [x] Optional: `cli_integration_test` if present
- Test coverage target: >=80% for `internal/udsapi`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/udsapi`
- `make verify` passes

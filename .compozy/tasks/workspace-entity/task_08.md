---
status: completed
title: ACP AdditionalDirs for workspace roots
type: backend
complexity: medium
dependencies:
  - task_04
---

# Task 08: ACP AdditionalDirs for workspace roots

## Overview

Extend ACP `StartOpts` and JSON-RPC payloads so agent processes receive `AdditionalDirs` for multi-root workspaces: primary `cmd.Dir` remains workspace root; extra directories are forwarded per TechSpec for `session/new` and `session/load` flows.

<critical>
- READ `_techspec.md` ACP impact and `internal/acp/types.go`
- TESTS REQUIRED — extend `client_test.go` / handler tests
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST add `AdditionalDirs []string` to `StartOpts` (or equivalent) with validation (absolute dirs, exist)
- MUST include additional dirs in RPC to agent for both start and load session paths
- MUST normalize paths consistently with Resolver `ResolvedWorkspace` (root + `AdditionalDirs`)
- MUST update `normalizeStartOpts` and tests for invalid additional dirs
- MUST implement the JSON-RPC contract per TechSpec; GREENFIELD — sem suporte a payloads antigos ou formatos duplicados só por compat
</requirements>

## Subtasks
- [x] 8.1 Extend `StartOpts` and validation in `normalizeStartOpts`
- [x] 8.2 Plumb `AdditionalDirs` through `Driver.Start` and load-session RPC
- [x] 8.3 Update mock/helper processes in tests to assert dirs received
- [x] 8.4 Document wire field names in ACP handler code comments

## Implementation Details

See TechSpec "internal/acp/" row and Development Sequencing step 11. Follow existing JSON-RPC patterns in `internal/acp/client.go`.

### Relevant Files
- `internal/acp/types.go` — `StartOpts`
- `internal/acp/client.go` — `Start`, `normalizeStartOpts`, RPC methods
- `internal/acp/client_test.go` — Process helpers

### Dependent Files
- `internal/session/manager.go` — Supplies opts from `ResolvedWorkspace`

## Deliverables
- `AdditionalDirs` end-to-end from session to subprocess RPC
- Updated unit tests in `internal/acp/` **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `Start` with empty `AdditionalDirs` unchanged behavior vs baseline
  - [x] `Start` with multiple additional dirs includes them in RPC payload / helper stdin as designed
  - [x] Invalid additional dir (non-absolute, missing) fails `normalizeStartOpts` / `Validate`
- Integration tests:
  - [x] `client_integration_test.go` reviewed; no start-payload assertions required changes
- Test coverage target: >=80% for `internal/acp`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/acp`
- `make verify` passes

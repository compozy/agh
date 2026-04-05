---
status: completed
title: Memory API & CLI
type: ""
complexity: high
dependencies:
    - task_01
    - task_04
---

# Task 05: Memory API & CLI

## Overview

Expose the memory system through all three interfaces: HTTP API (web UI), UDS API (CLI IPC), and CLI commands. This adds 5 memory endpoints to both HTTP and UDS servers (list, read, write, delete, consolidate) and 5 CLI subcommands under `agh memory`. Agents interact with memory via CLI passthrough through ACP `terminal/create` calls — this is the primary write path for agent-driven memory creation.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "API Endpoints" for exact routes, request/response types, and error mapping
- REFERENCE TECHSPEC "Data Models > CLI commands" for subcommand signatures and scope resolution
- REFERENCE `.old_project/internal/kernel/api.go` for the old project's memory HTTP handlers
- REFERENCE `.old_project/internal/cli/memory.go` for the old project's CLI command patterns
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add 5 memory routes to HTTP server: `GET /api/memory`, `GET /api/memory/:filename`, `PUT /api/memory/:filename`, `DELETE /api/memory/:filename`, `POST /api/memory/consolidate`
- MUST add same 5 routes to UDS server (mirrors HTTP)
- MUST implement scope resolution: when scope not specified, infer from memory type (user/feedback → global, project/reference → workspace)
- MUST implement error mapping: `os.ErrNotExist` → 404, validation errors → 400, all other → 500
- MUST accept `scope` and `workspace` as query parameters for GET/DELETE, body fields for PUT/POST
- MUST add `MemoryStore` (or equivalent interface) to `RuntimeDeps` for handler access
- MUST add `DreamService` (or equivalent trigger interface) to `RuntimeDeps` for consolidation trigger
- MUST add 5 CLI subcommands: `agh memory list`, `read`, `write`, `delete`, `consolidate`
- MUST add daemon client methods in `cli/client.go` for all 5 memory API calls
- MUST support `--scope global|workspace` flag on list/read/delete commands
- MUST support `--type` and `--description` required flags on write command
- MUST support `--content` flag or stdin for write command content
- MUST support human and JSON output formats for list and read commands
- MUST register `memory` subcommand in `root.go` alongside existing session/agent/observe commands
- MUST follow existing Cobra CLI patterns (factory functions, RunE, flag binding)
- MUST extend `GET /api/observe/health` response with memory stats: `global_files`, `workspace_files`, `last_consolidation`, `dream_enabled`
- MUST follow existing Gin handler patterns (request binding, JSON response, error handling)
</requirements>

## Subtasks
- [x] 5.1 Add memory handler file to httpapi with 5 endpoint handlers and request/response types
- [x] 5.2 Add memory handler file to udsapi with 5 endpoint handlers (mirrors httpapi)
- [x] 5.3 Add `MemoryStore` and `DreamTrigger` to `RuntimeDeps` and wire in daemon
- [x] 5.4 Register memory routes in both HTTP and UDS server constructors
- [x] 5.5 Add daemon client methods in `cli/client.go` for memory API calls
- [x] 5.6 Implement `agh memory` command group with 5 subcommands in `cli/memory.go`
- [x] 5.7 Register memory command in `cli/root.go`

## Implementation Details

New files:
- `internal/httpapi/memory.go` — Memory HTTP handlers and request/response types
- `internal/udsapi/memory.go` — Memory UDS handlers (or add to existing handlers.go)
- `internal/cli/memory.go` — `agh memory` command group with 5 subcommands

Modified files:
- `internal/daemon/daemon.go` — Add Store and DreamService to RuntimeDeps
- `internal/httpapi/server.go` — Register memory routes
- `internal/udsapi/server.go` or `routes.go` — Register memory routes
- `internal/cli/root.go` — Register memory subcommand
- `internal/cli/client.go` — Add memory API client methods

Reference TechSpec "API Endpoints" table for exact routes and types. Follow the patterns in existing handler files (`httpapi/sessions.go`, `udsapi/handlers.go`, `cli/session.go`).

### Relevant Files
- `.old_project/internal/kernel/api.go` — Old memory HTTP handlers (list, read, write, delete, consolidate)
- `.old_project/internal/cli/memory.go` — Old CLI commands with scope resolution and formatters
- `internal/httpapi/sessions.go` — Existing handler patterns to follow (request binding, response format)
- `internal/httpapi/server.go:57-82` — Server struct and route registration
- `internal/udsapi/handlers.go` — Existing UDS handler patterns (combined handler file)
- `internal/udsapi/routes.go` — Route registration function
- `internal/cli/session.go` — Existing CLI command patterns (Cobra factory, flags, RunE)
- `internal/cli/client.go` — Existing daemon client methods to extend
- `internal/cli/root.go:74-80` — Where subcommands are registered
- `internal/cli/format.go` — Output formatting patterns (human, JSON)
- `internal/memory/store.go` (task_01) — Store API used by handlers

### Dependent Files
- None — this is the final task in the chain

### Related ADRs
- [ADR-001: Interleaved Extensibility](adrs/adr-001.md) — Memory exposed via standard API, not plugin system

## Deliverables
- `internal/httpapi/memory.go` with 5 endpoint handlers
- `internal/udsapi/memory.go` with 5 endpoint handlers
- `internal/cli/memory.go` with 5 CLI subcommands
- Modified `internal/cli/client.go` with memory client methods
- Modified `internal/cli/root.go` with memory command registration
- Modified `internal/daemon/daemon.go` with RuntimeDeps additions
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for API endpoints **(REQUIRED)**

## Tests
- Unit tests (HTTP/UDS handlers):
  - [x] `GET /api/memory` returns list of MemoryHeaders as JSON
  - [x] `GET /api/memory?scope=global` filters to global scope
  - [x] `GET /api/memory?scope=workspace&workspace=/path` filters to workspace scope
  - [x] `GET /api/memory/:filename` returns memory file content
  - [x] `GET /api/memory/:filename` with non-existent file returns 404
  - [x] `PUT /api/memory/:filename` with valid content creates/updates file
  - [x] `PUT /api/memory/:filename` with invalid frontmatter returns 400
  - [x] `PUT /api/memory/:filename` with missing content returns 400
  - [x] `DELETE /api/memory/:filename` removes file and returns 200
  - [x] `DELETE /api/memory/:filename` with non-existent file returns 404
  - [x] `POST /api/memory/consolidate` triggers dream and returns `{triggered: true}`
  - [x] `POST /api/memory/consolidate` when gates fail returns `{triggered: false, reason: "..."}`
  - [x] `GET /api/observe/health` includes memory stats (global_files, workspace_files, last_consolidation, dream_enabled)
  - [x] Scope resolution: type=user without scope defaults to global
  - [x] Scope resolution: type=project without scope defaults to workspace
- Unit tests (CLI commands):
  - [x] `agh memory list` outputs formatted header list
  - [x] `agh memory list --scope global` passes scope to API
  - [x] `agh memory read <filename>` outputs file content
  - [x] `agh memory write <filename> --type user --description "desc"` sends PUT with content
  - [x] `agh memory write` with `--content` flag uses flag value
  - [x] `agh memory delete <filename>` sends DELETE request
  - [x] `agh memory consolidate` sends POST and displays result
  - [x] JSON output format works for list and read commands
- Integration tests:
  - [x] Full round-trip: write via API → read via API → verify content matches
  - [x] Full round-trip: write via CLI → list via CLI → shows new memory
  - [x] Delete via API removes file from subsequent list calls
  - [x] Consolidate endpoint interacts with dream Service correctly
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing (new + existing)
- Test coverage >=80% on new code
- `make verify` passes
- All 5 memory endpoints accessible via HTTP and UDS
- All 5 CLI subcommands functional with human and JSON output
- `agh memory --help` shows usage for all subcommands

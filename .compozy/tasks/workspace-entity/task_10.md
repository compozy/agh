---
status: completed
title: HTTP API workspace routes and session contract
type: backend
complexity: high
dependencies:
  - task_04
  - task_06
---

# Task 10: HTTP API workspace routes and session contract

## Overview

Add the `/api/workspaces` REST group (CRUD, list, resolve), change `POST /api/sessions` and `GET /api/sessions` to the workspace-aware contract in the TechSpec, and inject `workspace.Resolver` into HTTP handlers. Update handler tests and OpenAPI or docstrings if the project uses them.

<critical>
- READ `_techspec.md` "API Endpoints" tables
- TESTS REQUIRED — `handlers_test.go`, route list tests
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST implement all workspace routes: POST/GET/PATCH/DELETE, list, detail with related sessions/agents/skills summary as specified
- MUST implement `POST /api/workspaces/resolve` for path-based resolve/register
- MUST require workspace reference for session creation (`workspace` id/name and/or `workspace_path`) per TechSpec
- MUST add `GET /api/sessions?workspace=` filter
- MUST return appropriate HTTP status codes for duplicate name/path, missing workspace, missing root
- MUST update `internal/httpapi/helpers_test.go` and `handlers_test.go` fixtures
</requirements>

## Subtasks
- [x] 10.1 Add Gin route group and handler structs with Resolver dependency
- [x] 10.2 Implement request/response JSON types matching TechSpec field names
- [x] 10.3 Update session create/list handlers and validation
- [x] 10.4 Expand tests for new routes and updated session payloads
- [x] 10.5 Ensure SSE or existing streams include `workspace_id` where session metadata is exposed

## Implementation Details

See TechSpec "API Endpoints". Follow patterns in `internal/httpapi/` for binding, errors, and tests. Session tests today use `{"workspace":"/workspace"}` — migrate to new shape.

### Relevant Files
- `internal/httpapi/` — Router setup, handlers, helpers
- `internal/httpapi/handlers_test.go` — Route enumeration and handler tests
- `internal/httpapi/helpers_test.go` — Shared test fixtures

### Dependent Files
- `web/` — Consumes new API (task_13)
- `internal/udsapi/` — Mirrors HTTP (task_11)

## Deliverables
- Full workspace HTTP surface + updated session API
- Comprehensive handler tests **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `POST /api/workspaces` with new root registers workspace and returns `ws_` id
  - [x] `GET /api/workspaces` lists registered rows
  - [x] `PATCH /api/workspaces/:id` updates name and additional dirs
  - [x] `DELETE /api/workspaces/:id` returns 204
  - [x] `POST /api/workspaces/resolve` with absolute path returns workspace json
  - [x] `POST /api/sessions` rejects body with neither workspace nor workspace_path
  - [x] `GET /api/sessions?workspace=ws_xxx` filters correctly
- Integration tests:
  - [x] Covered by httptest in handlers if sufficient
- Test coverage target: >=80% for `internal/httpapi`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/httpapi`
- `make verify` passes

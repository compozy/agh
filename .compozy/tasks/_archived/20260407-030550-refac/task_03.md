---
status: completed
title: API layer consolidation (apicore + apitest)
type: refactor
complexity: critical
dependencies:
  - task_02
---

# Task 03: API layer consolidation (apicore + apitest)

## Overview

Create `internal/apicore/` as the shared foundation for `httpapi/` and `udsapi/`, then consolidate test infrastructure into `internal/apitest/`. This is the highest-impact refactoring — eliminating ~900 lines of duplicated production code and ~1,700 lines of duplicated test code. After this task, each transport package is a thin binding layer over the shared core.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
**apicore — interfaces:**
- MUST define shared `SessionManager` (13 methods including `ApprovePermission`), `Observer`, `DreamTrigger`, `WorkspaceService` interfaces in `apicore/`
- MUST resolve the `ApprovePermission` gap (udsapi currently has 12 methods vs httpapi's 13)

**apicore — payloads + conversions:**
- MUST move all payload structs to `apicore/` as single source of truth
- MUST handle `agentEventPayload.Timestamp` divergence: base uses `time.Time`, httpapi keeps AI SDK variant with `string`
- MUST move all `*FromInfo`, `*FromEvent`, `*FromDef`, `payloadJSON` conversion functions

**apicore — parsers + SSE + errors:**
- MUST move query parsers: `parseSessionEventQuery`, `parseOptionalTime`, `parseOptionalInt`, `parseOptionalInt64`, `parseObserveEventQuery`, `parseObserveCursor`
- MUST move SSE infrastructure: `prepareSSE`, `writeSSE`, `writeSSERaw`, `emitObserveEvents`, `observeEventAfterCursor`, `observeEventID`, `flushWriter`
- MUST create shared `RespondError(c, status, err, maskInternalErrors bool)` preserving httpapi (masks 5xx) vs udsapi (exposes raw) divergence

**apicore — handlers:**
- MUST create `BaseHandlers` struct with shared dependencies
- MUST move shared handler methods: `listSessions`, `createSession`, `getSession`, `stopSession`, `resumeSession`, `sessionEvents`, `sessionHistory`, `sessionTranscript`, `listAgents`, `getAgent`, `observeEvents`, `daemonStatus`, `health`
- MUST move all memory handlers verbatim (byte-identical between packages)
- MUST move all workspace handlers verbatim (byte-identical between packages)
- Transport packages MUST delegate shared logic through `BaseHandlers` or equivalent composition
- httpapi-specific MUST remain: AI SDK streaming, static files, CORS, `approveSession`
- udsapi-specific MUST remain: raw streaming, socket lifecycle, `approveSession` (501 stub)

**apitest:**
- MUST create `internal/apitest/` with shared `StubSessionManager`, `StubObserver`, `StubWorkspaceService`
- MUST move shared test helpers: `parseSSE`, `performRequest`, `decodeJSONResponse`, `newTestHomePaths`, `writeAgentDef`, `newSessionInfo`, `newSession`, `discardLogger`
- Transport-specific helpers MUST stay: `mustStaticFS` (httpapi), `shortSocketPath`/`newUnixClient` (udsapi)
- SHOULD consolidate identical test cases for shared handlers

**General:**
- MUST NOT introduce circular dependencies
- MUST NOT change any HTTP/UDS API contract (request/response JSON shapes)
- MUST keep transport route registration intact while binding the shared handler implementations from the consolidated core
</requirements>

## Subtasks

- [x] 3.1 Create `apicore/interfaces.go` with shared interfaces
- [x] 3.2 Create `apicore/payloads.go` with all request/response structs + `apicore/conversions.go`
- [x] 3.3 Create `apicore/parsers.go` + `apicore/sse.go` + `apicore/errors.go`
- [x] 3.4 Create `apicore/handlers.go` with `BaseHandlers` + shared session/agent/observe/daemon handlers
- [x] 3.5 Create `apicore/memory.go` + `apicore/workspaces.go` with shared handlers
- [x] 3.6 Update `httpapi/` and `udsapi/` to reuse `BaseHandlers` via embedding or composition, remove all duplicated code
- [x] 3.7 Create `internal/apitest/` with shared stubs and helpers, update both test suites

## Implementation Details

See TechSpec "Phase 3: API Layer Consolidation" steps 3.1–3.9 for the full extraction strategy. See [API report](./20260406-api-layer.md) F1–F5 for the complete duplication inventory with line-by-line mapping.

### Relevant Files

**Interfaces (to be unified):**
- `internal/httpapi/server.go:41-77` — SessionManager (13 methods), Observer, DreamTrigger, WorkspaceService
- `internal/udsapi/server.go:40-75` — same interfaces minus `ApprovePermission`

**Payloads (to be extracted):**
- `internal/httpapi/sessions.go:17-59` — session payloads
- `internal/httpapi/agents.go:14-30` — agent payloads
- `internal/httpapi/prompt.go:30-61` — event payloads (Timestamp as `string`)
- `internal/httpapi/observe.go:11-18` — observe payload
- `internal/httpapi/daemon.go:11-21` — daemon payload
- `internal/httpapi/stream.go:21-39` — error/SSE payloads
- `internal/httpapi/memory.go:19-52` — memory payloads
- `internal/httpapi/workspaces.go:19-50` — workspace payloads
- `internal/udsapi/handlers.go:57-194` — all duplicate payloads (post-split: in `payloads.go`)

**Handlers (to be moved):**
- `internal/httpapi/sessions.go:67-200` — session handlers
- `internal/httpapi/agents.go:32-113` — agent handlers
- `internal/httpapi/observe.go:20-69` — observe handlers
- `internal/httpapi/daemon.go:23-40` — daemon handler
- `internal/httpapi/memory.go:54-433` — memory handlers (byte-identical)
- `internal/httpapi/workspaces.go:52-279` — workspace handlers (byte-identical)
- `internal/httpapi/server.go:530-571` — `newHandlers` constructor
- `internal/httpapi/server.go:470-528` — `RegisterRoutes`
- `internal/udsapi/routes.go:6-60` — `RegisterRoutes`

**SSE + parsers + errors:**
- `internal/httpapi/stream.go:289-376` — SSE infra + respondError (masks 5xx)
- `internal/httpapi/sessions.go:225-290` — query parsers
- `internal/udsapi/handlers.go:700-1058` — duplicate parsers, SSE, conversions, respondError (exposes raw)

**Test infrastructure:**
- `internal/httpapi/helpers_test.go:30-408` — stubs + helpers (408 lines)
- `internal/udsapi/helpers_test.go:29-374` — duplicate stubs + helpers (374 lines)
- `internal/httpapi/helpers_test.go:252-261` — `mustStaticFS` (stays)
- `internal/udsapi/helpers_test.go:248-256` — `shortSocketPath` (stays)
- `internal/udsapi/helpers_test.go:360-370` — `newUnixClient` (stays)
- `internal/httpapi/handlers_test.go` (840 lines) — handler tests
- `internal/udsapi/handlers_test.go` (862 lines) — ~90% duplicate tests

### Dependent Files

- `internal/httpapi/server.go` — `Handlers` reuses `apicore.BaseHandlers`
- `internal/udsapi/server.go` — `Handlers` reuses `apicore.BaseHandlers`
- `internal/daemon/daemon.go` — may need minor updates to handler construction
- All httpapi + udsapi handler files — remove duplicated code, import apicore
- Both `helpers_test.go` — import apitest, keep only transport-specific helpers

## Deliverables

- `internal/apicore/` package (~8-9 files): interfaces, payloads, conversions, parsers, sse, errors, handlers, memory, workspaces
- `internal/apitest/` package: shared stubs and test helpers
- Reduced `httpapi/` — only transport-specific code remains
- Reduced `udsapi/` — only transport-specific code remains
- Consolidated test suites
- Unit tests with >=80% coverage for apicore **(REQUIRED)**
- `make verify` passes **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `SessionPayloadFromInfo` correctly converts all fields
  - [x] `AgentPayloadFromDef` correctly converts agent definitions
  - [x] `ParseSessionEventQuery` parses valid params and rejects invalid
  - [x] `ParseOptionalTime` handles empty, valid RFC3339, and invalid format
  - [x] `RespondError` with `maskInternalErrors=true` masks 5xx details
  - [x] `RespondError` with `maskInternalErrors=false` exposes raw error
  - [x] `PrepareSSE` sets correct headers
  - [x] `BaseHandlers.listSessions` returns sessions from SessionManager
  - [x] `BaseHandlers.createSession` calls Create with correct opts
  - [x] `BaseHandlers.getSession` returns 404 for unknown ID
  - [x] Memory handlers delegate to memory store correctly
  - [x] Workspace handlers delegate to workspace service correctly
  - [x] `apitest.StubSessionManager` satisfies `apicore.SessionManager`
- Integration tests:
  - [x] All existing `httpapi/handlers_test.go` assertions pass
  - [x] All existing `udsapi/handlers_test.go` assertions pass
  - [x] All memory and workspace handler tests pass
- Test coverage target: >=80%

## Success Criteria

- All tests passing
- Test coverage >=80% for `apicore/`
- `make verify` passes
- No duplicated handler methods between `httpapi/` and `udsapi/`
- No duplicated payload structs (except httpapi AI SDK variant)
- No duplicated SSE utilities, parsers, or conversion functions
- No duplicated test stubs between packages
- `httpapi/` is reduced primarily to transport/server concerns plus HTTP-only behavior (AI SDK streaming, static assets, CORS, approval route wiring)
- `udsapi/` is reduced primarily to transport/server concerns plus UDS-only behavior (socket lifecycle, raw streaming, approval stub wiring)
- `golangci-lint` confirms no import cycles

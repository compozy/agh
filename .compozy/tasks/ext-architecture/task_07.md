---
status: completed
title: Host API handler (bidirectional JSON-RPC)
type: backend
complexity: high
dependencies:
  - task_04
  - task_06
---

# Task 07: Host API handler (bidirectional JSON-RPC)

## Overview

Implement the Host API handler that processes JSON-RPC requests from extensions calling back into AGH. Extensions invoke methods like `sessions/create`, `memory/store`, and `observe/events` to drive AGH workflows. Every call is capability-checked against the extension's negotiated grants per ADR-003 and the protocol spec section 5.2. The handler bridges the subprocess transport into AGH's existing session manager, memory store, observer, and skills registry.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/extension/host_api.go` with `HostAPIHandler` struct
- MUST implement all Host API methods from `_protocol.md` section 5.2: `sessions/list`, `sessions/create`, `sessions/prompt`, `sessions/stop`, `sessions/status`, `sessions/events`, `memory/recall`, `memory/store`, `memory/forget`, `observe/health`, `observe/events`, `skills/list`
- MUST enforce capability grants using the `CapabilityChecker` before executing any method (both granted_actions method-level AND granted_security family-level)
- MUST return typed JSON-RPC errors per `_protocol.md` section 9: `-32001 capability_denied`, `-32002 rate_limited`, `-32601 method_not_found`
- MUST implement per-extension rate limiting with typed `-32002` error including `retry_after_ms`
- MUST delegate method execution to existing AGH services: `session.Manager`, `memory.Store`, `observe.Observer`, `skills.Registry`
- MUST NOT expose any AGH internals not listed in the protocol spec Host API inventory
- MUST include `since?` parameter in `observe/events` per protocol spec
- MUST handle unknown method names with `-32601 method_not_found` per protocol spec section 9.3
</requirements>

## Subtasks
- [x] 7.1 Create `HostAPIHandler` struct with dependencies on session manager, memory store, observer, skills registry
- [x] 7.2 Implement request dispatcher that maps method names to handler functions
- [x] 7.3 Implement `sessions/*` method handlers (list, create, prompt, stop, status, events)
- [x] 7.4 Implement `memory/*` method handlers (recall, store, forget)
- [x] 7.5 Implement `observe/*` and `skills/*` method handlers
- [x] 7.6 Implement per-extension rate limiting with typed error responses
- [x] 7.7 Write unit and integration tests covering all methods and error paths

## Implementation Details

New file `internal/extension/host_api.go` and `internal/extension/host_api_test.go`.

See TechSpec "Host API" section for the method inventory. See `_protocol.md` section 5.2 for the canonical table with capability requirements. See `_protocol.md` section 9 for error codes.

The handler is transport-agnostic — it receives method + params and returns result + error. The `Manager` (task 06) wires this into the subprocess transport layer so each extension's inbound requests land in the correct `HostAPIHandler` invocation with the extension name attached for capability checks.

### Relevant Files
- `internal/extension/capability.go` — `CapabilityChecker.CheckHostAPI()` enforcement (task 04)
- `internal/extension/manager.go` — Manager provides extension context and routes inbound requests (task 06)
- `internal/session/manager.go` — Session service methods the Host API delegates to
- `internal/memory/store.go` — Memory service the Host API delegates to
- `internal/observe/observer.go` — Observer for health and events queries
- `internal/skills/registry.go` — Skills registry for skills/list
- `internal/api/httpapi/` — Existing HTTP handler pattern for similar method dispatch

### Dependent Files
- `internal/extension/manager.go` — Manager wires HostAPIHandler into subprocess inbound message routing (task 06 already prepared)
- `internal/daemon/boot.go` — Will wire HostAPIHandler with real dependencies at boot (task 08)

### Related ADRs
- [ADR-003: Capability-Scoped Security Model](adrs/adr-003.md) — Every handler call is capability-checked
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Host API is the "actions" dimension

## Deliverables
- New `internal/extension/host_api.go` with `HostAPIHandler` struct and all methods
- Per-extension rate limiting implementation
- Typed error responses matching `_protocol.md` error codes
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with a real session manager and memory store **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `sessions/list` returns authorized sessions for extension with `session.read`
  - [x] `sessions/list` returns `-32001 capability_denied` without `session.read`
  - [x] `sessions/create` returns new session ID with `session.write`
  - [x] `sessions/create` returns `-32001` without `session.write`
  - [x] `sessions/prompt` delivers message to session with correct turn ID
  - [x] `sessions/stop` terminates the session
  - [x] `sessions/status` returns session state for authorized extensions
  - [x] `sessions/events` returns event stream with optional `since` parameter
  - [x] `memory/store` persists content with tags
  - [x] `memory/recall` returns ranked matches
  - [x] `memory/forget` removes entries
  - [x] `observe/health` returns daemon health snapshot
  - [x] `observe/events` returns filtered events with `since` parameter
  - [x] `skills/list` returns skills for workspace
  - [x] Unknown method returns `-32601 method_not_found`
  - [x] Rate limit exceeded returns `-32002` with `retry_after_ms` in data
  - [x] All methods return typed error data with method name and required capabilities
- Integration tests:
  - [x] Extension creates session via Host API → session runs → extension reads events back
  - [x] Extension stores memory then recalls it
  - [x] Unauthorized extension attempts all methods → all return capability denied
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- All 12 Host API methods implemented and tested
- Capability enforcement validated for every method
- `make verify` passes

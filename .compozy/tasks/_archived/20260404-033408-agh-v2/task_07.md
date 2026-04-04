---
status: completed
domain: API
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_04
  - task_05
  - task_06
---

# Task 07: UDS API Package

## Overview

Implement the `internal/udsapi` package — the Unix Domain Socket server that exposes the daemon's session Manager to the CLI. Uses Gin with a unix socket listener. Supports both request/response and SSE streaming (for `--follow` and `wait` commands).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use Gin framework with unix socket listener
- MUST expose same endpoints as HTTP API (shared route definitions where possible)
- MUST support JSON request/response for standard operations
- MUST support SSE streaming over UDS for `--follow` and `wait` commands
- MUST expose session management: create, list, stop, status, resume, prompt, events, history
- MUST expose agent management: list, info
- MUST expose observability: events query, health
- MUST expose daemon status
- MUST handle clean socket file creation and cleanup
</requirements>

## Subtasks
- [x] 7.1 Implement UDS server with Gin and unix socket listener
- [x] 7.2 Implement session endpoints (create, list, stop, status, resume, prompt, events, history, stream)
- [x] 7.3 Implement agent endpoints (list, info)
- [x] 7.4 Implement observe endpoints (events, events/stream, health)
- [x] 7.5 Implement daemon status endpoint
- [x] 7.6 Implement SSE streaming over UDS for follow/wait patterns
- [x] 7.7 Implement graceful shutdown with connection draining

## Implementation Details

Create the following files:
- `internal/udsapi/server.go` — UDS server setup, unix socket listener, lifecycle
- `internal/udsapi/routes.go` — Route registration (shared with httpapi where possible)
- `internal/udsapi/handlers.go` — Request handlers calling session Manager and observe

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — API Endpoints table, SSE Streaming Contracts, UDS API section

### Old Project Reference
- `.old_project/internal/transport/uds.go` — HTTP over unix socket implementation

### Related ADRs
- [ADR-003: ACP Internally, HTTP/SSE Externally](../adrs/adr-003.md) — UDS for CLI, HTTP for web
- [ADR-009: Agent-First Observability](../adrs/adr-009.md) — CLI-queryable via UDS

## Deliverables
- `internal/udsapi/` package with full UDS server
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with real unix socket **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Route registration covers all endpoints from TechSpec
  - [x] Session create handler: valid request returns session ID
  - [x] Session list handler: returns all sessions
  - [x] Session prompt handler: returns SSE stream
  - [x] Session events handler: returns filtered events
  - [x] Agent list handler: returns available agents
  - [x] Health handler: returns metrics
  - [x] Daemon status handler: returns running state
- Integration tests:
  - [x] Full round-trip: client → UDS → handler → session Manager → response
  - [x] SSE streaming over UDS: subscribe, receive events, disconnect
  - [x] Server starts on socket path, client connects, exchanges messages
  - [x] Graceful shutdown: in-flight requests complete, then server stops
  - [x] SSE reconnection: Last-Event-ID resumes from correct sequence (parity with HTTP API)
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- CLI can connect to daemon via UDS and execute all commands
- SSE streaming works over UDS for --follow and wait patterns

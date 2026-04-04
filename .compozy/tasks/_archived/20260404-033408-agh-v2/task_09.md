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

# Task 09: HTTP API Package

## Overview

Implement the `internal/httpapi` package — the Gin HTTP/SSE server that exposes the daemon to the web UI. Implements all REST endpoints plus three SSE streaming contracts (prompt-scoped, session-wide, cross-session). Compatible with Vercel AI SDK `useChat` for the future web UI.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use Gin framework
- MUST implement all HTTP endpoints from TechSpec API table
- MUST implement three SSE streaming contracts:
  - `POST /api/sessions/:id/prompt` — prompt-scoped (ends on completion)
  - `GET /api/sessions/:id/stream` — session-wide (long-lived, supports Last-Event-ID)
  - `GET /api/observe/events/stream` — cross-session (long-lived, supports Last-Event-ID)
- MUST support `Last-Event-ID` header for SSE reconnection/replay
- MUST include sequence numbers in SSE events for resumption
- SSE format MUST be compatible with Vercel AI SDK `x-vercel-ai-ui-message-stream: v1` for prompt endpoint
- MUST implement permission approval endpoint (`POST /api/sessions/:id/approve`) for future interactive flow
- MUST implement structured JSON error responses
- MUST implement request logging middleware
- MUST support graceful shutdown with connection draining
</requirements>

## Subtasks
- [x] 9.1 Implement Gin server setup with middleware (logging, error handling, CORS)
- [x] 9.2 Implement session REST endpoints (create, list, get, stop, resume, events, history)
- [x] 9.3 Implement session prompt endpoint with SSE streaming
- [x] 9.4 Implement session stream endpoint (long-lived SSE with Last-Event-ID)
- [x] 9.5 Implement agent endpoints (list, info)
- [x] 9.6 Implement observe endpoints (events, events/stream, health)
- [x] 9.7 Implement daemon status endpoint
- [x] 9.8 Implement permission approval endpoint
- [x] 9.9 Implement graceful shutdown

## Implementation Details

Create the following files:
- `internal/httpapi/server.go` — Gin server setup, middleware, lifecycle
- `internal/httpapi/sessions.go` — Session REST endpoints
- `internal/httpapi/prompt.go` — Prompt SSE streaming (AI SDK compatible)
- `internal/httpapi/stream.go` — Session and cross-session SSE streams
- `internal/httpapi/agents.go` — Agent endpoints
- `internal/httpapi/observe.go` — Observability endpoints
- `internal/httpapi/daemon.go` — Daemon status endpoint

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — API Endpoints table, SSE Streaming Contracts

### Old Project Reference
- `.old_project/internal/dashboard/server.go` — Gin HTTP server setup and routing
- `.old_project/internal/dashboard/api.go` — REST endpoint implementations
- `.old_project/internal/dashboard/websocket.go` — Real-time streaming patterns
- `.old_project/internal/dashboard/events.go` — Event broadcasting patterns

### Related ADRs
- [ADR-003: ACP Internally, HTTP/SSE Externally](../adrs/adr-003.md) — HTTP/SSE for web clients
- [ADR-009: Agent-First Observability](../adrs/adr-009.md) — Event streaming endpoints

## Deliverables
- `internal/httpapi/` package with full HTTP/SSE server
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with real HTTP server **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Route registration covers all endpoints from TechSpec
  - [x] Session create: valid request returns session ID with 201
  - [x] Session create: missing agent returns 400
  - [x] Session list: returns all sessions with correct format
  - [x] Session stop: returns 200, session state becomes stopped
  - [x] Agent list: returns available agents from config
  - [x] Health: returns correct metrics structure
  - [x] Error responses: consistent JSON error format
- Integration tests:
  - [x] Full HTTP round-trip: POST create → GET list → POST prompt (SSE) → GET events
  - [x] SSE prompt stream: receives agent_message, tool_call, done events
  - [x] SSE session stream: receives events with sequence IDs
  - [x] SSE reconnection: Last-Event-ID resumes from correct sequence
  - [x] CORS headers present on responses
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All endpoints return correct responses per TechSpec
- SSE streaming delivers events in real-time
- Last-Event-ID reconnection works correctly
- Prompt endpoint format compatible with AI SDK expectations

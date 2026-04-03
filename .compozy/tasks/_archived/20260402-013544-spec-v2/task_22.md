---
status: completed
domain: Dashboard
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_03
    - task_05
    - task_06
---

# Task 22: Web Server & WebSocket

## Overview
Implement the HTTP server that serves the embedded dashboard frontend, provides REST API endpoints for topology/agents/workgroups/blackboard data, and handles WebSocket connections for live PTY output streaming and real-time topology updates.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use Gin as the HTTP framework for both UDS and TCP listeners
- MUST serve embedded static files via go:embed for the dashboard SPA per docs/spec-v2/16-web-dashboard.md
- MUST implement REST endpoints: GET /api/topology, GET /api/agents, GET /api/agents/{id}, GET /api/workgroups, GET /api/workgroups/{id}, GET /api/blackboard per docs/spec-v2/08-data-models.md
- MUST implement WebSocket endpoint /ws/pty/{agent-id} for PTY output streaming per docs/spec-v2/16-web-dashboard.md
- MUST implement WebSocket endpoint /ws/topology for real-time topology updates per docs/spec-v2/16-web-dashboard.md
- MUST use nhooyr.io/websocket for WebSocket server
- MUST replay ring buffer contents on PTY WebSocket connect (scrollback)
- MUST send full topology snapshot on topology WebSocket connect
- MUST implement TopologyBroadcaster for fan-out to all topology subscribers
- MUST implement WsHub for per-agent PTY subscriber management
- MUST handle backpressure: buffer up to 64KB per client, drop oldest on overflow
- MUST be configurable (enabled/disabled, host, port) per docs/spec-v2/07-configuration.md
- MUST use UDS listener at ~/.agh/daemon.sock for CLI-to-kernel communication (same Gin router)
- MUST use TCP listener on configurable port (default 2123) for dashboard access
- MUST serve the same Gin router on both UDS and TCP listeners

**Dependencies:** gin-gonic/gin
</requirements>

## Subtasks
- [x] 22.1 Implement Gin HTTP server with go:embed static file serving, dual listeners (UDS at ~/.agh/daemon.sock + TCP for dashboard)
- [x] 22.2 Implement REST endpoints for topology, agents, workgroups, blackboard
- [x] 22.3 Implement WebSocket /ws/pty/{agent-id} with ring buffer replay and live streaming
- [x] 22.4 Implement WebSocket /ws/topology with snapshot and incremental updates
- [x] 22.5 Implement WsHub for PTY subscriber management and fan-out
- [x] 22.6 Implement TopologyBroadcaster for topology event distribution
- [x] 22.7 Implement backpressure handling and graceful WebSocket shutdown

## Implementation Details
Refer to docs/spec-v2/16-web-dashboard.md for the complete WebSocket protocol, REST API, and ring buffer behavior. Refer to docs/spec-v2/08-data-models.md for JSON response schemas.

### Relevant Files
- `docs/spec-v2/16-web-dashboard.md` — dashboard server spec
- `docs/spec-v2/08-data-models.md` — REST/WebSocket endpoint catalog, JSON schemas
- `docs/spec-v2/02-kernel.md` — WebSocket server, WsHub

### Dependent Files
- `internal/pty/` — ring buffer access, subscriber registration
- `internal/registry/` — agent/workgroup data for REST responses
- `internal/state/` — SQLite queries for blackboard endpoint

## Deliverables
- internal/dashboard/server.go — Gin HTTP server with go:embed, dual listeners (UDS + TCP), and routing
- internal/dashboard/api.go — REST endpoint handlers
- internal/dashboard/websocket.go — WebSocket handlers for PTY and topology
- internal/dashboard/events.go — TopologyBroadcaster, WsHub
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for REST and WebSocket **(REQUIRED)**

## Tests
- Unit tests:
  - [x] GET /api/topology returns correct JSON with workgroups and agents
  - [x] GET /api/agents returns flat list of all agents
  - [x] GET /api/agents/{id} returns single agent or 404
  - [x] GET /api/workgroups returns all workgroups with hierarchy
  - [x] GET /api/blackboard returns entries filtered by scope
  - [x] WsHub tracks subscribers per agent correctly
  - [x] WsHub removes subscriber on disconnect
  - [x] TopologyBroadcaster fans events to all topology subscribers
- Integration tests:
  - [x] PTY WebSocket: connect → receive ring buffer scrollback → receive live data
  - [x] PTY WebSocket: multiple clients receive same output simultaneously
  - [x] Topology WebSocket: connect → receive snapshot → receive incremental updates
  - [x] Topology WebSocket: reconnect receives fresh snapshot
  - [x] UDS listener at ~/.agh/daemon.sock accepts CLI HTTP requests
  - [x] TCP listener serves dashboard on configurable port
  - [x] Same Gin router serves both UDS and TCP listeners
  - [x] Dashboard disabled config: no TCP server started (UDS still active for CLI)
  - [x] Graceful shutdown: WebSocket close frames sent to all clients
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- REST responses match JSON schemas from docs/spec-v2/08-data-models.md
- WebSocket protocol matches docs/spec-v2/16-web-dashboard.md
- Gin router serves both UDS (~/.agh/daemon.sock) and TCP (dashboard port)

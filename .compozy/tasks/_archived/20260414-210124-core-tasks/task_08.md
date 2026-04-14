---
status: completed
title: "Expose task and run routes through HTTP and UDS"
type: backend
complexity: medium
dependencies:
  - task_07
---

# Task 08: Expose task and run routes through HTTP and UDS

## Overview
Expose the new task domain consistently through both daemon transports. This task should add the transport-specific routes and wiring needed to make the shared core handlers reachable via HTTP/SSE clients and the local UDS-backed CLI.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. HTTP and UDS MUST expose the same task and run capabilities with parity in route coverage and request semantics.
2. Transport wiring MUST remain thin and delegate request handling to the shared core handlers introduced in the previous task.
3. The new routes MUST preserve existing daemon transport organization rather than introducing a parallel routing style for tasks.
</requirements>

## Subtasks
- [x] 8.1 Add task and run route groups to the HTTP API router.
- [x] 8.2 Add matching task and run route groups to the UDS API router.
- [x] 8.3 Extend HTTP and UDS server configuration to accept the task handler dependency.
- [x] 8.4 Ensure response envelopes, status codes, and path parameters match across both transports.

## Implementation Details
Use the TechSpec "API Surface" section as the route inventory. Follow the route-group organization already used for automation and network surfaces in both `httpapi` and `udsapi`.

### Relevant Files
- `internal/api/httpapi/routes.go` — Existing HTTP route registration patterns.
- `internal/api/httpapi/handlers.go` — Existing HTTP handler wiring and dependency injection patterns.
- `internal/api/httpapi/server.go` — Existing server dependency wiring patterns.
- `internal/api/udsapi/routes.go` — Existing UDS route registration patterns.
- `internal/api/udsapi/server.go` — Existing UDS server dependency wiring patterns.
- `internal/api/core/handlers.go` — Shared handler entrypoints to expose here.

### Dependent Files
- `internal/cli/task.go` — Will call the UDS routes introduced here.
- `internal/daemon/boot.go` — Will need to inject the task handler dependency into both servers.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Determines which lifecycle endpoints are valid to expose.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Requires channel filters and fields to survive transport routing intact.

## Deliverables
- HTTP routes for task and run operations.
- UDS routes for task and run operations with parity to HTTP.
- Transport wiring that injects the task core handlers into both servers.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for transport parity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Verify route registration covers the expected task and run paths in both HTTP and UDS routers.
  - [x] Verify server construction fails fast when task handlers are missing from required transport configuration.
- Integration tests:
  - [x] Verify the same create/list/get/update task flows succeed through both HTTP and UDS surfaces.
  - [x] Verify run lifecycle endpoints behave identically through both transports for the same manager behavior.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Tasks and runs are reachable through both daemon transports with matching semantics
- Transport packages remain thin adapters over the shared handler layer

---
status: completed
title: Network CLI/API surface and observability
type: backend
complexity: medium
dependencies:
  - task_07
---

# Task 08: Network CLI/API surface and observability

## Overview

Expose the network runtime through shared contracts, UDS handlers, CLI commands, and observability/status surfaces. This task gives users and agents a stable control-plane interface for sending messages, listing peers and spaces, and inspecting runtime health.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add shared contract payloads for network send, status, peers, spaces, inbox, and audit-related responses
- MUST expose UDS routes and handlers for the network control plane without bypassing daemon-owned validation
- MUST add CLI commands for `agh network status`, `peers`, `spaces`, `send`, and `inbox`, including machine-readable output where the spec requires it
- MUST surface structured logs, metrics, and daemon status fields needed to observe the runtime
- MUST preserve and expose correlation fields and AGH workflow/handoff `ext` metadata when present, without making them required for v0 operation
</requirements>

## Subtasks
- [x] 8.1 Add shared network DTOs and conversion helpers in the API contract layer
- [x] 8.2 Implement UDS routes and handlers for network control-plane actions
- [x] 8.3 Add CLI commands and client methods for network status, discovery, send, and inbox operations
- [x] 8.4 Extend observability and status reporting for network lifecycle and queue metrics

## Implementation Details

The CLI and UDS layers should only talk through contracts and daemon handlers. They must not reach into transport or router internals directly.
Observability output should make multi-hop debugging possible by preserving `reply_to`, `trace_id`, `causation_id`, and optional AGH `ext` workflow/handoff keys when they are available in runtime data.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - API endpoints, CLI expectations, and monitoring sections
- `internal/api/contract/contract.go` - Add shared network request and response payloads
- `internal/api/core/handlers.go` - Add conversion helpers and handler plumbing for network results
- `internal/api/udsapi/routes.go` - Register network endpoints
- `internal/api/udsapi/network.go` - New network handler implementations
- `internal/cli/client.go` - Add client methods for the new network API surface
- `internal/cli/network.go` - New CLI command tree for network operations
- `internal/observe/observer.go` - Extend metrics and structured event reporting

### Dependent Files
- `internal/skills/bundled/skills/agh-network/SKILL.md` - Bundled skill examples depend on stable CLI semantics
- `internal/session/manager_start.go` - Prompt injection will reference the CLI surface exposed here
- `internal/daemon/info.go` - Status reporting may need updated network fields

### Related ADRs
- [ADR-003: CLI + Bundled Skill for Agent Network Communication](adrs/adr-003.md) - Establishes CLI as the outbound control plane
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - APIs consume daemon-owned runtime services

## Deliverables
- Shared contract and UDS/CLI implementations for the network control plane
- Observability and status updates for network health and queue state
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for CLI -> UDS -> daemon network flows **(REQUIRED)**

## Tests
- Unit tests:
- [x] Contract payload conversions preserve network request and response semantics
- [x] UDS handlers validate required arguments and report structured errors
- [x] CLI output formatting supports human-readable and machine-readable use cases
- [x] Observability surfaces include the expected network metrics and log events
- [x] Optional workflow/handoff metadata remains visible in status/audit surfaces without being treated as mandatory protocol state
- Integration tests:
- [x] CLI commands can list peers/spaces, send messages, and inspect inbox/status through the daemon
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Users and agents can operate the network runtime through stable CLI and UDS interfaces
- Network health and delivery behavior are visible through daemon status and observability surfaces

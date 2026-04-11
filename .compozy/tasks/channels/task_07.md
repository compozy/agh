---
status: completed
title: Compose channel manager and wire daemon boot lifecycle
type: backend
complexity: high
dependencies:
  - task_03
  - task_05
  - task_06
---

# Task 07: Compose channel manager and wire daemon boot lifecycle

## Overview

Compose the channel subsystem into the daemon so the registry, Host API, delivery broker, and channel-capable extensions can run as one managed runtime. This task owns startup, shutdown, restart, and dependency injection for the new channel substrate without breaking the current `daemon/` composition rules.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a composed channel runtime to the daemon that wires the channel registry, outbound target resolver, Host API channel methods, and delivery broker together.
2. MUST manage channel extension lifecycle in a way that preserves daemon-owned route continuity across start, stop, and restart flows.
3. MUST resolve and inject bound channel-instance secret material at startup through a daemon-owned resolver rather than through extension-initiated secret reads.
4. SHOULD keep all cross-package composition inside `internal/daemon/` and avoid introducing back-pointers from core packages into the daemon.
</requirements>

## Subtasks
- [x] 7.1 Add a channel manager/runtime composition entry point under `internal/channels/` or `internal/daemon/`
- [x] 7.2 Wire the new runtime into daemon boot and shutdown lifecycle paths
- [x] 7.3 Connect instance startup, restart, and stop flows to extension manager lifecycle handling
- [x] 7.4 Add tests for daemon composition, lifecycle transitions, and route continuity across restarts

## Implementation Details

Follow the TechSpec sections "System Architecture", "Data flow", "Impact Analysis", and "Development Sequencing". This task should wire the runtime and lifecycle only; it should not expose API routes or CLI commands yet.

### Relevant Files
- `internal/daemon/boot.go` — The daemon boot sequence is where the new channel runtime must be composed and started
- `internal/daemon/composed_assembler.go` — Existing composition-root wiring patterns belong here
- `internal/daemon/extensions.go` — Extension runtime composition and helper patterns already live here
- `internal/daemon/boundary.go` — Shared daemon-owned service boundaries may need to expand for channels
- `internal/extension/manager.go` — Channel instance lifecycle will rely on existing extension-process management behavior

### Dependent Files
- `internal/api/httpapi/routes.go` — Transport handlers later depend on the daemon exposing channel management services
- `internal/cli/root.go` — CLI command wiring later depends on the daemon/runtime composition introduced here
- `internal/observe/health.go` — Observability later consumes daemon-owned channel runtime state

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Requires a daemon-owned substrate composed alongside the existing extension system
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — The daemon runtime must preserve route continuity across extension restarts
- [ADR-008: Bound Secret Injection per Channel Instance](adrs/adr-008.md) — Startup orchestration must resolve bound secrets from the daemon side

## Deliverables
- Composed channel runtime and lifecycle wiring in `internal/daemon/`
- Startup, shutdown, and restart orchestration for channel instances and channel-capable extensions
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for daemon lifecycle and route continuity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Daemon composition builds the channel runtime only when its dependencies are present and valid
  - [x] Channel instance startup resolves bound secret material through the daemon-owned resolver before launching the extension
  - [x] Disabling or stopping a channel instance prevents new ingress while preserving prior route data
  - [x] Restart logic reuses daemon-owned routing state instead of rebuilding it from extension-local storage
- Integration tests:
  - [x] Starting the daemon with one configured channel instance launches the channel-capable extension and exposes a ready channel runtime
  - [x] Restarting the extension process preserves route continuity and allows later delivery to resume through the same channel instance
  - [x] Daemon shutdown drains the channel runtime cleanly without leaving active delivery goroutines behind
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The channel subsystem starts and stops as a first-class daemon runtime component
- Route continuity remains daemon-owned across channel extension restarts

## Verification Evidence
- `go test -coverprofile=/tmp/daemon.cover ./internal/daemon` → passed, `80.0%` statement coverage for `internal/daemon`
- `go test -tags integration ./internal/daemon` → passed
- `make verify` → passed

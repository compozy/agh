---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_03
    - task_04
    - task_05
    - task_06
    - task_08
    - task_13
---

# Task 14: Kernel Boot & Shutdown Orchestration

## Overview
Wire all kernel subsystems into the central Kernel struct and implement the daemon boot sequence. The kernel boots as a daemon process that owns shared infrastructure (NATS, UDS, HTTP, config, roles, drivers) but does NOT create sessions at boot. Sessions are created on demand via `agh session start`. Implement the oklog/run group for goroutine lifecycle, signal handling via signal.NotifyContext, and cascading graceful shutdown (sessions in parallel first, then kernel infra).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- REFERENCE docs/plans/2026-03-30-multi-session-design.md for daemon architecture
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement the Kernel struct that owns global shared infrastructure per docs/plans/2026-03-30-multi-session-design.md
- MUST implement the daemon boot sequence:
  1. Acquire daemon.lock (gofrs/flock) at ~/.agh/daemon.lock — exit with error if another daemon is running
  2. Write daemon.json at ~/.agh/daemon.json with PID, socket path, start time
  3. Parse global config (~/.agh/config.toml)
  4. Init logger
  5. Start NATS broker (DontListen: true)
  6. Create UDS listener at ~/.agh/daemon.sock (HTTP over UDS with Gin)
  7. Load RoleCatalog from global ~/.agh/roles/
  8. Init DriverRegistry
  9. Load prompt templates
  10. Init SessionManager (empty)
  11. Start HTTP server (Gin, TCP for dashboard)
  12. Start signal handler
- MUST NOT create sessions at boot — kernel boots empty and waits for `agh session start`
- MUST use oklog/run for top-level goroutine orchestration
- MUST use signal.NotifyContext for SIGINT/SIGTERM handling
- MUST implement cascading graceful shutdown: stop accepting CLI commands, stop all sessions in parallel, wait with timeout, close WebSocket, stop HTTP (Gin), drain NATS, close NATS, remove ~/.agh/daemon.sock, remove ~/.agh/daemon.json, release daemon.lock
- MUST provide NewKernel constructor with functional options pattern (WithDriver, WithConfig, etc.)
- MUST log "kernel ready, listening on ~/.agh/daemon.sock and :2123" on successful boot
- MUST initialize SessionManager as empty map at boot

**Dependencies:** gofrs/flock, gin-gonic/gin
</requirements>

## Subtasks
- [x] 14.1 Implement Kernel struct owning NATS, UDS, HTTP (Gin), Config, RoleCatalog, DriverRegistry, PromptTemplates, Logger, SessionManager, flock
- [x] 14.2 Implement daemon boot sequence: acquire flock → write daemon.json → load global config → init logger → start NATS → create UDS (Gin) → load roles → init drivers → init SessionManager → start HTTP (Gin) → start signal handler
- [x] 14.3 Implement oklog/run group with all kernel goroutines
- [x] 14.4 Implement signal handling and cascading shutdown (sessions in parallel → kernel infra → release flock → remove daemon.json → remove daemon.sock)
- [x] 14.5 Implement NewKernel constructor with functional options
- [x] 14.6 Integrate with SessionManager (empty at boot, sessions added via task_15)

## Implementation Details
Refer to docs/plans/2026-03-30-multi-session-design.md for the daemon boot sequence and cascading shutdown. Refer to docs/spec-v2/01-architecture.md for component interaction. Refer to docs/spec-v2/09-resilience.md for graceful shutdown steps.

### Relevant Files
- `docs/plans/2026-03-30-multi-session-design.md` — daemon architecture, boot sequence, shutdown
- `docs/spec-v2/01-architecture.md` — component interaction
- `docs/spec-v2/09-resilience.md` — graceful shutdown
- `docs/spec-v2/11-testing.md` — bootTestKernel helper pattern

### Dependent Files
- All internal/ packages from previous tasks
- `internal/kernel/types.go` — Kernel struct (added in task_08)
- `internal/session/` — SessionManager (implemented in task_15)

## Deliverables
- internal/kernel/kernel.go — Kernel struct, NewKernel, boot, shutdown
- Full boot-to-shutdown integration tests
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for boot and shutdown **(REQUIRED)**

## Tests
- Unit tests:
  - [x] NewKernel creates Kernel with all global subsystems initialized
  - [x] Functional options (WithDriver, WithConfig) correctly configure kernel
  - [x] Boot sequence: acquires daemon.lock, writes daemon.json, initializes NATS, UDS (Gin), HTTP (Gin), RoleCatalog, DriverRegistry in order
  - [x] Boot fails with clear error if daemon.lock already held by another process
  - [x] SessionManager is empty after boot (no sessions created)
  - [x] Kernel logs readiness message after boot ("listening on ~/.agh/daemon.sock and :2123")
  - [x] Global config loaded from ~/.agh/config.toml
  - [x] Global roles loaded from ~/.agh/roles/
- Integration tests:
  - [x] Full kernel boot with mock driver: NATS running, UDS accepting at ~/.agh/daemon.sock, HTTP serving (Gin), no sessions
  - [x] Signal handling triggers cascading shutdown
  - [x] Shutdown stops all sessions in parallel before stopping kernel infra
  - [x] Shutdown cleans up: remove ~/.agh/daemon.sock, remove ~/.agh/daemon.json, release daemon.lock
  - [x] HTTP server (Gin) stops gracefully with timeout
  - [x] NATS broker closed after all sessions stopped
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Kernel boots as daemon with no sessions, UDS at ~/.agh/daemon.sock
- Kernel shuts down cleanly with cascading session stop, releases flock, removes daemon.json and daemon.sock
- bootTestKernel helper pattern works for all subsequent tests

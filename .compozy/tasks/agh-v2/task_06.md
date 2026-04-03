---
status: pending
domain: Infrastructure
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_01
  - task_02
  - task_03
  - task_04
  - task_05
---

# Task 06: Daemon Package

## Overview

Implement the `internal/daemon` package — the sole composition root that wires all other packages together, manages the daemon lock file, orchestrates the boot sequence and graceful shutdown, and handles signal processing. This is the only package that imports all other internal packages.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST be the sole composition root — the only package importing all others
- MUST implement daemon lock file (`~/.agh/daemon.lock`) via `gofrs/flock`
- MUST detect and handle stale locks (check if PID is still running)
- MUST write `daemon.json` with PID, port, started_at on boot
- MUST clean up stale `daemon.sock` on boot
- MUST implement boot sequence: load config → acquire lock → open global DB → create providers → create session manager → start servers → run reconciliation
- MUST implement graceful shutdown: stop sessions → stop HTTP → stop UDS → close DBs → release lock
- MUST handle SIGINT and SIGTERM for graceful shutdown
- MUST wire the Notifier (observe/) to session manager
- MUST run boot-time reconciliation via observe/
- MUST use functional options pattern for New()
- MUST implement Run(ctx) that blocks until signal or ctx cancel
</requirements>

## Subtasks
- [ ] 6.1 Implement daemon lock acquisition with stale PID detection
- [ ] 6.2 Implement daemon.json write/read/cleanup
- [ ] 6.3 Implement boot sequence: config → lock → global DB → providers → session manager → Notifier wiring
- [ ] 6.4 Implement graceful shutdown with ordered teardown
- [ ] 6.5 Implement signal handling (SIGINT, SIGTERM)
- [ ] 6.6 Implement Run() that starts all servers and blocks
- [ ] 6.7 Wire reconciliation on boot
- [ ] 6.8 Implement orphan agent process cleanup on daemon restart (scan for processes whose parent PID matches stale daemon)
- [ ] 6.9 Add `Boundaries()` verification to boot sequence (optional: log warnings if import violations detected in dev mode)

## Implementation Details

Create the following files:
- `internal/daemon/daemon.go` — Daemon struct, New(), Run(), Shutdown()
- `internal/daemon/lock.go` — Lock file management, stale detection
- `internal/daemon/info.go` — DaemonInfo read/write (daemon.json)

Also create the entry point:
- `cmd/agh/main.go` — CLI binary that delegates to daemon/ for start command

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — Daemon section, Failure Handling, Boot sequence

### Old Project Reference
- `.old_project/internal/kernel/kernel.go` — Boot sequence and composition root (what to simplify)
- `.old_project/internal/cli/daemon.go` — Daemon start/stop, lock file handling
- `.old_project/internal/kernel/dream/lock.go` — PID-based lock file patterns

### Related ADRs
- [ADR-002: Pragmatic Flat Architecture](../adrs/adr-002.md) — daemon/ as sole composition root

## Deliverables
- `internal/daemon/` package with Daemon struct, boot, shutdown, lock
- `cmd/agh/main.go` entry point
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for boot/shutdown lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Lock acquisition succeeds when no lock exists
  - [ ] Lock acquisition fails when another daemon holds the lock
  - [ ] Stale lock detection: dead PID detected, lock re-acquired
  - [ ] DaemonInfo: write and read back correctly
  - [ ] Stale socket cleanup on boot
  - [ ] Shutdown: ordered teardown (sessions first, then servers, then DBs)
- Integration tests:
  - [ ] Full boot sequence: lock → DB → session manager → ready
  - [ ] Graceful shutdown via context cancellation
  - [ ] Signal handling: SIGINT triggers shutdown
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Daemon boots, acquires lock, writes daemon.json
- Second daemon instance fails with lock error
- Graceful shutdown cleans up all resources
- Stale locks from crashed daemons are recovered

---
status: pending
title: Wire Hooks in daemon — replace notifierFanout
type: refactor
complexity: critical
dependencies:
  - task_06
  - task_07
  - task_08
---

# Task 9: Wire Hooks in daemon — replace notifierFanout

## Overview

Hard cut-over in the daemon composition root: delete `notifierFanout` and `skillsHookDispatcher`, wire the new `Hooks` struct as the `session.Notifier`, connect reload triggers from the skills watcher, and update the shutdown sequence to `stop sessions → Hooks.Close() → close servers → close DB`. This is the most critical integration point — it connects the entire hooks platform to the running daemon.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST delete `notifierFanout` struct and all its methods from `daemon/notifier.go`
- MUST delete `skillsHookDispatcher` struct and all its methods from `daemon/notifier.go`
- MUST delete `sessionHookPhase` interface from `daemon/notifier.go`
- MUST create `Hooks` in `boot.go` using `hooks.NewHooks()` with functional options
- MUST register Go-native hooks, config declarations, and skill declarations into `Hooks`
- MUST pass `Hooks` as `session.Notifier` to session manager
- MUST wire `observe.Observer` as a notifier within `Hooks` (or compose separately — preserve existing observability)
- MUST connect skills watcher reload to `Hooks.Rebuild()`
- MUST connect dream handler callback (postSessionStopped) to `Hooks` via native hook or direct callback
- MUST update shutdown sequence: stop sessions → `Hooks.Close()` → shutdown HTTP → shutdown UDS → close DB → release lock
- MUST remove `hookRunner` creation from boot.go
</requirements>

## Subtasks
- [ ] 9.1 Delete `notifierFanout`, `skillsHookDispatcher`, `sessionHookPhase` from `daemon/notifier.go`
- [ ] 9.2 Create and configure `Hooks` in `boot.go` with all declaration providers
- [ ] 9.3 Wire `Hooks` as `session.Notifier` in session manager creation
- [ ] 9.4 Wire skills watcher to trigger `Hooks.Rebuild()` on change
- [ ] 9.5 Migrate dream handler and observer callbacks to work with `Hooks`
- [ ] 9.6 Update shutdown sequence in `daemon.go`
- [ ] 9.7 Write integration tests for daemon wiring and hot reload

## Implementation Details

Modify existing files:
- `internal/daemon/notifier.go` — Delete almost entire file, keep only minimal content or delete file entirely
- `internal/daemon/boot.go` — Replace notifier composition (lines 197-273) with Hooks creation and wiring
- `internal/daemon/daemon.go` — Update shutdown sequence (lines 369-442) to include Hooks.Close()

Reference TechSpec "Migration from Current Hooks Implementation" and "Async Worker Pool" (shutdown ordering) sections.

### Relevant Files
- `internal/daemon/notifier.go:21-148` — notifierFanout and skillsHookDispatcher to delete
- `internal/daemon/boot.go:122` — hookRunner creation to delete
- `internal/daemon/boot.go:197-273` — Notifier composition to rewrite
- `internal/daemon/daemon.go:369-442` — Shutdown sequence to update
- `internal/hooks/hooks.go` (task_06) — Hooks struct to instantiate
- `internal/session/interfaces.go:150-155` — Notifier interface that Hooks implements
- `internal/observe/observer.go` — Observer that was previously wired into notifierFanout

### Dependent Files
- `internal/session/` — Receives new Notifier (no code change needed — same interface)
- `internal/observe/` — May need adjustment if observer was composed in notifierFanout

### Related ADRs
- [ADR-013: Hot-Reloadable Registry with RWMutex Snapshot Swap](../adrs/adr-013.md) — Dispatcher replaces notifierFanout

## Deliverables
- Deleted/gutted `daemon/notifier.go`
- Rewritten notifier composition in `boot.go`
- Updated shutdown sequence in `daemon.go`
- Integration tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `Hooks` is created with valid options and starts pool
  - [ ] `Hooks` is wired as session.Notifier — compile-time check
  - [ ] Shutdown sequence calls `Hooks.Close()` after session stop
- Integration tests:
  - [ ] Daemon boot creates `Hooks`, registers declarations, builds initial registry
  - [ ] Skills watcher file change triggers `Hooks.Rebuild()` — new hooks visible in next dispatch
  - [ ] Session create fires `session.post_create` hooks via `Hooks.OnSessionCreated`
  - [ ] Session stop fires `session.post_stop` hooks via `Hooks.OnSessionStopped`
  - [ ] Graceful shutdown drains async hooks before closing database
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt, lint, test, build)
- Zero references to `notifierFanout` or `skillsHookDispatcher` in codebase
- Daemon boots, runs sessions, and shuts down cleanly with new Hooks

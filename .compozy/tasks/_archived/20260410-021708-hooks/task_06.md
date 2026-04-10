---
status: completed
title: Hooks struct with typed dispatch, registry, and Notifier
type: backend
complexity: high
dependencies:
  - task_04
  - task_05
---

# Task 6: Hooks struct with typed dispatch, registry, and Notifier

## Overview

Assemble the main `Hooks` struct that owns the hot-reloadable registry (RWMutex + snapshot swap), the async worker pool, and exposes typed dispatch functions for every event in the taxonomy. The `Hooks` struct also implements `session.Notifier` to serve as the replacement for the current `notifierFanout`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define `Hooks` struct owning: `sync.RWMutex`, snapshot map `map[HookEvent][]*ResolvedHook`, async worker pool, version counter `atomic.Int64`
- MUST implement `Rebuild(ctx)` — builds complete snapshot from all 4 sources, validates, sorts, swaps under write lock. Skip swap if unchanged.
- MUST implement read path: `RLock`, copy slice reference, `RUnlock`, dispatch against snapshot
- MUST implement typed dispatch functions for all 27 events (one function per event) using `pipeline[P, R]`
- MUST implement `session.Notifier` interface: `OnSessionCreated`, `OnSessionStopped`, `OnAgentEvent`
- MUST implement `Close()` for graceful async pool shutdown
- MUST provide compile-time interface check: `var _ session.Notifier = (*Hooks)(nil)`
- MUST use functional options pattern for constructor: `NewHooks(opts ...Option)`
</requirements>

## Subtasks
- [x] 6.1 Define `Hooks` struct with registry, pool, mutex, version counter, and observer dependencies
- [x] 6.2 Implement `NewHooks(opts ...Option)` constructor with functional options
- [x] 6.3 Implement `Rebuild(ctx)` with build-then-validate-then-swap semantics
- [x] 6.4 Implement typed dispatch functions for all event families using pipeline
- [x] 6.5 Implement `session.Notifier` interface bridging to typed dispatch
- [x] 6.6 Implement `Close()` for pool shutdown
- [x] 6.7 Write unit tests for registry swap, dispatch, and Notifier implementation

## Implementation Details

Create new files in `internal/hooks/`:
- `hooks.go` — Main Hooks struct, constructor, Close, Rebuild
- `dispatch.go` — All typed dispatch functions
- `notifier.go` — session.Notifier implementation

Follow `internal/skills/registry.go` pattern for RWMutex + snapshot swap (lines 28+, `reloadGlobal`). Reference TechSpec "Hot Reload" and "Core Interfaces" sections.

### Relevant Files
- `internal/hooks/pipeline.go` (task_04) — Pipeline engine for sync execution
- `internal/hooks/pool.go` (task_05) — Async worker pool
- `internal/hooks/ordering.go` (task_02) — Sorts hooks for registry snapshot
- `internal/hooks/normalize.go` (task_02) — Validates declarations
- `internal/skills/registry.go:28-120` — Existing RWMutex + snapshot swap pattern
- `internal/session/interfaces.go:150-155` — Notifier interface to implement

### Dependent Files
- `internal/daemon/boot.go` — Will wire Hooks in task_09
- `internal/session/` — Will receive Hooks as Notifier in task_09

### Related ADRs
- [ADR-005: Use Typed Per-Event Dispatch Functions](../adrs/adr-005.md) — Typed dispatch function design
- [ADR-013: Hot-Reloadable Registry with RWMutex Snapshot Swap](../adrs/adr-013.md) — Registry design

## Deliverables
- `internal/hooks/hooks.go` with Hooks struct, constructor, Rebuild, Close
- `internal/hooks/dispatch.go` with all typed dispatch functions
- `internal/hooks/notifier.go` with session.Notifier implementation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `NewHooks()` creates Hooks with empty registry and started pool
  - [x] `Rebuild()` with valid declarations populates registry snapshot
  - [x] `Rebuild()` with invalid declaration keeps old snapshot and returns error
  - [x] `Rebuild()` with unchanged declarations skips swap (version unchanged)
  - [x] `Rebuild()` bumps version counter on successful swap
  - [x] Concurrent `Rebuild()` and dispatch do not race (test with `-race`)
  - [x] Typed dispatch function returns original payload when no hooks match
  - [x] Typed dispatch function applies patches from matching hooks in order
  - [x] `OnSessionCreated` calls the appropriate dispatch function
  - [x] `OnSessionStopped` calls the appropriate dispatch function
  - [x] `Close()` drains async pool and returns
  - [x] Compile-time check: `var _ session.Notifier = (*Hooks)(nil)` compiles
- Integration tests:
  - [x] Full dispatch with native + subprocess hooks on same event — ordering correct
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `session.Notifier` compile-time check passes
- Registry swap is atomic — no partial state visible to readers
- `-race` flag passes with concurrent rebuild + dispatch

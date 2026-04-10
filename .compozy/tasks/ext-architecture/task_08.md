---
status: completed
title: Daemon boot integration
type: backend
complexity: medium
dependencies:
  - task_06
---

# Task 08: Daemon boot integration

## Overview

Wire the Extension Manager into AGH's daemon composition root. Add a new boot phase between the hooks system initialization and the servers startup. The Extension Manager must be initialized with real dependencies (session manager, memory store, observer, skills registry), register extension-provided hook declarations into the existing hooks rebuild cycle, and participate in the graceful shutdown sequence.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add a new boot phase in `internal/daemon/boot.go` between `bootHooks()` and `bootServers()`
- MUST initialize `extension.Manager` with real dependencies: session manager, memory store, observer, skills registry, extension registry from global DB
- MUST wire the extension declaration provider into the existing hooks `DeclarationProvider` chain via `hooks_bridge.go`
- MUST trigger `hooks.Rebuild()` after Extension Manager starts so extension-provided hooks are dispatched
- MUST add Extension Manager to the daemon shutdown cleanup chain (LIFO order) so extensions are stopped before servers
- MUST NOT break any existing daemon boot tests
- MUST handle extension manager start failure gracefully (log and continue boot — extensions are not critical)
- MUST emit log events at each extension lifecycle transition (loaded, failed, shutdown)
</requirements>

## Subtasks
- [x] 8.1 Add `bootExtensions()` phase in `internal/daemon/boot.go` between `bootHooks` and `bootServers`
- [x] 8.2 Wire `extension.Manager` with real session manager, memory, observer, skills dependencies
- [x] 8.3 Extend `internal/daemon/hooks_bridge.go` with extension declaration provider
- [x] 8.4 Trigger `hooks.Rebuild()` after extension manager starts
- [x] 8.5 Add extension manager Stop() to LIFO cleanup chain
- [x] 8.6 Write integration tests validating extension boot phase and shutdown order

## Implementation Details

Modify `internal/daemon/boot.go` and `internal/daemon/hooks_bridge.go`. No new files — this is pure integration work.

See TechSpec "Integration Points / Daemon Composition Root" section for the boot phase location and wiring pattern.

The new `bootExtensions` phase follows the existing `bootXxx()` pattern: takes a context, returns a cleanup function, logs its progress. Extension Manager failure should NOT block daemon boot — log and continue with zero extensions loaded.

### Relevant Files
- `internal/daemon/boot.go` — Boot phase sequence; add new `bootExtensions()` phase
- `internal/daemon/hooks_bridge.go` — Existing `daemonNativeHooks()` and declaration provider patterns
- `internal/daemon/daemon.go` — `Daemon` struct and option pattern
- `internal/extension/manager.go` — Manager to initialize (task 06)
- `internal/hooks/hooks.go` — `hooks.Rebuild()` and `DeclarationProvider` pattern

### Dependent Files
- `internal/daemon/daemon_test.go` — Existing daemon tests that must continue passing
- `internal/extension/manager.go` — Will be the consumer of the wired dependencies

### Related ADRs
- [ADR-001: Two-Tier Extension Model](adrs/adr-001.md) — Establishes that extensions are wired at daemon boot composition

## Deliverables
- Modified `internal/daemon/boot.go` with new `bootExtensions` phase
- Extended `internal/daemon/hooks_bridge.go` with extension declaration provider
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for full daemon boot with extensions enabled **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `bootExtensions()` initializes Manager with correct dependencies
  - [x] `bootExtensions()` returns no-op cleanup when no extensions installed
  - [x] Extension Manager start failure logs error but does not fail boot
  - [x] Extension declaration provider returns declarations from loaded extensions
  - [x] `hooks.Rebuild()` is called after Extension Manager starts
  - [x] Shutdown order: Extension Manager stops before servers
- Integration tests:
  - [x] Full daemon boot with one test extension → extension loads → hooks rebuild → extension stops on daemon shutdown
  - [x] Daemon boot with corrupt extension → logs error, continues boot, other extensions load normally
  - [x] Extension provides hook declarations → hook dispatches route to extension correctly
- Test coverage target: >=80%
- All existing daemon tests continue passing

## Success Criteria
- All tests passing
- Test coverage >=80%
- Daemon boots successfully with zero, one, and multiple extensions
- Shutdown order verified via integration test
- `make verify` passes

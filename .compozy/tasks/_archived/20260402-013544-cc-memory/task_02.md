---
status: completed
title: Dream Consolidation Package
type: ""
complexity: high
dependencies:
    - task_01
---

# Task 02: Dream Consolidation Package

## Overview

Implement the `internal/kernel/dream/` package that provides background memory consolidation through a 3-gate triggering system, PID-based lock file with mtime-as-state, and a `DreamService` that spawns ephemeral agent sessions for 4-phase memory synthesis. This package enables cross-session learning by distilling session transcripts and blackboard entries into durable memory files.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `ConsolidationLock` with PID body, mtime-as-lastConsolidatedAt, stale detection (dead PID OR >1h), and rollback
- MUST implement `TryAcquire()` with race-condition handling (write-then-verify pattern)
- MUST implement `Rollback()` that rewinds mtime to pre-acquire state on failure
- MUST implement 3-gate `ShouldRun()` ordered cheapest-first: time gate → session gate → lock gate
- Time gate: hours since `lastConsolidatedAt` (from lock mtime) > configurable threshold (default 24h)
- Session gate: completed sessions since last consolidation >= configurable threshold (default 3)
- Lock gate: `TryAcquire()` succeeds
- MUST implement `DreamService` with functional options pattern
- MUST implement `Run()` that calls a `SessionSpawner` callback with goal and consolidation prompt
- MUST define 4-phase consolidation prompt template (orient/gather/consolidate/prune) as embedded markdown
- MUST use `syscall.Kill(pid, 0)` for PID liveness check (not process table lookup)
- `DreamService` MUST NOT import kernel or session packages — uses `SessionSpawner` callback for decoupling
</requirements>

## Subtasks

- [x] 2.1 Implement `ConsolidationLock` with acquire, release, stale detection, and rollback
- [x] 2.2 Implement PID liveness check and race-condition safe acquisition
- [x] 2.3 Implement `ShouldRun()` with 3-gate evaluation in cheapest-first order
- [x] 2.4 Implement session directory scanning for session gate (count sessions since last consolidation)
- [x] 2.5 Implement `DreamService` struct with functional options (`WithMinHours`, `WithMinSessions`, `WithLogger`, etc.)
- [x] 2.6 Create 4-phase consolidation prompt template as embedded markdown
- [x] 2.7 Implement `Run()` method that invokes `SessionSpawner` callback
- [x] 2.8 Write comprehensive unit tests for lock, gates, and service

## Implementation Details

Create new package at `internal/kernel/dream/` with the following files:

- `lock.go` — `ConsolidationLock` struct, `TryAcquire()`, `Release()`, `Rollback()`, `LastConsolidatedAt()`
- `dream.go` — `DreamService` struct, functional options, `ShouldRun()`, `Run()`, `SessionSpawner` type
- `prompt.go` — 4-phase consolidation prompt template (embedded string or go:embed)
- `lock_test.go` — unit tests for lock operations
- `dream_test.go` — unit tests for gate evaluation and service construction

### Relevant Files

- `internal/kernel/memdir/` (task_01) — `memdir.Store` used by DreamService to access memory directories
- `internal/kernel/kernel.go:197-416` — Boot sequence pattern for `NewKernel()` (reference for functional options)
- `internal/kernel/types.go` — Reference for how functional options are defined in the kernel

### Dependent Files

- `internal/kernel/kernel.go` (task_04) — Will initialize `DreamService` at boot and wire `SessionSpawner`
- `internal/kernel/session_manager.go` (task_04) — Will call `ShouldRun()` on session stop

### Related ADRs

- [ADR-002: Dream Consolidation via Ephemeral Agent Session](../adrs/adr-002.md) — Defines the session-spawning approach and 3-gate triggering
- [ADR-004: time.Ticker Over Cron Library](../adrs/adr-004.md) — Defines scheduling approach (ticker lives in kernel, not in this package)

## Deliverables

- `internal/kernel/dream/` package with lock, service, and prompt modules
- Consolidation lock with mtime-as-state and stale PID detection
- 3-gate triggering system with cheapest-first evaluation
- 4-phase consolidation prompt template
- Decoupled `SessionSpawner` callback type
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Lock acquire on fresh file succeeds, PID is written
  - [x] Lock acquire when already held by live process fails
  - [x] Lock acquire when held by dead process succeeds (stale PID reclaim)
  - [x] Lock acquire when older than 1 hour succeeds (age-based reclaim)
  - [x] Lock release clears PID
  - [x] Lock rollback rewinds mtime to prior value
  - [x] LastConsolidatedAt reads lock file mtime
  - [x] LastConsolidatedAt returns zero time for missing lock file
  - [x] ShouldRun returns false when time gate fails (too recent)
  - [x] ShouldRun returns false when session gate fails (too few sessions)
  - [x] ShouldRun returns false when lock gate fails (already held)
  - [x] ShouldRun returns true when all 3 gates pass
  - [x] ShouldRun evaluates gates in order (time → session → lock) — verify session dir not scanned when time gate fails
  - [x] DreamService construction with default options
  - [x] DreamService construction with custom options overrides defaults
  - [x] Run calls SessionSpawner with correct goal and prompt
  - [x] Run rolls back lock on SessionSpawner failure
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- Package has zero imports from `internal/kernel/` (decoupled via callback)
- Lock file handles concurrent acquirers safely
- Gate evaluation short-circuits on first failure

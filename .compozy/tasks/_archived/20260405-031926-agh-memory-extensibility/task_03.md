---
status: completed
title: Dream Consolidation Service
type: ""
complexity: medium
dependencies:
    - task_01
---

# Task 03: Dream Consolidation Service

## Overview

Implement the dream consolidation service within `internal/memory/`. This service periodically evaluates three gates (time elapsed, completed sessions, lock availability) and, when all pass, spawns an ephemeral ACP agent session to synthesize session transcripts into durable memory files. The consolidation lock uses a PID-file mechanism with mtime-as-timestamp for cross-process coordination. Ported from the cc-memory dream implementation, adapted to AGH v2's ACP session model.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Core Interfaces" for SessionSpawner callback type and Service API
- REFERENCE `.old_project/internal/kernel/dream/` for the proven implementation to port
- REFERENCE `.resources/claude-code/services/autoDream/` for Claude Code's Auto Dream patterns
- REFERENCE `.resources/openclaw/extensions/memory-core/src/dreaming.ts` for OpenClaw's dreaming presets
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `Service` struct with functional options pattern (`NewService(opts ...Option)`)
- MUST implement 3-gate evaluation in `ShouldRun()`: time gate, session gate, lock gate (evaluated in this order)
- MUST implement `ConsolidationLock` with PID-file body, mtime as `lastConsolidatedAt`, 1-hour stale age default
- MUST implement lock operations: `TryAcquire() (priorMtime, ok, error)`, `Release()`, `Rollback(priorMtime)`
- MUST implement stale lock reclamation via `syscall.Kill(pid, 0)` for dead PID detection
- MUST implement `Run(ctx, SessionSpawner)` that acquires lock, spawns session, and handles success/failure cleanup
- MUST implement session counting by scanning `meta.json` files in sessions directory for completed sessions since last consolidation — parse JSON independently (do NOT import `store/` package; memory/ imports only `config/` and stdlib per TechSpec)
- MUST define `SessionSpawner func(ctx, goal, prompt string) error` callback type (no import of session/)
- MUST embed the 4-phase consolidation prompt template (Orient, Gather, Consolidate, Prune)
- MUST provide functional options: `WithMemoryStore`, `WithSessionsDir`, `WithLockPath`, `WithMinHours`, `WithMinSessions`, `WithLogger`, `WithGoal`
- MUST use `errors.Join` for combining spawn error + rollback error
- MUST use `sync.Mutex` to prevent concurrent `Run()` calls within same process
- MUST NOT import `session/`, `daemon/`, `httpapi/`, `udsapi/`, or `cli/`
</requirements>

## Subtasks
- [x] 3.1 Implement `ConsolidationLock` with PID-file acquire/release/rollback and stale detection
- [x] 3.2 Implement session counting via `meta.json` scanning with time filter
- [x] 3.3 Implement `Service` struct with functional options and 3-gate `ShouldRun()`
- [x] 3.4 Implement `Run()` orchestration with lock management and spawner callback
- [x] 3.5 Create embedded 4-phase consolidation prompt template
- [x] 3.6 Write comprehensive tests for all gate combinations and lock edge cases

## Implementation Details

Add these files to `internal/memory/`:
- `lock.go` — `ConsolidationLock` struct with PID-file coordination
- `dream.go` — `Service` struct, gate evaluation, `Run()` orchestration, functional options
- `prompt.go` — Embedded consolidation prompt template (4-phase markdown)

Reference TechSpec "Dream Consolidation Prompt" section for the 4-phase template content. Reference TechSpec "Data Models > Consolidation lock file" for lock mechanics.

### Relevant Files
- `.old_project/internal/kernel/dream/dream.go` — Proven DreamService to port (gate logic, Run orchestration)
- `.old_project/internal/kernel/dream/lock.go` — Proven ConsolidationLock to port (PID, mtime, stale)
- `.old_project/internal/kernel/dream/prompt.md` — 4-phase consolidation prompt to adapt
- `.old_project/internal/kernel/dream/dream_test.go` — Test patterns and edge cases
- `.resources/claude-code/services/autoDream/` — Claude Code's Auto Dream trigger logic
- `.resources/openclaw/extensions/memory-core/src/dreaming.ts` — OpenClaw dreaming presets and promotion scoring
- `internal/memory/store.go` (task_01) — Store used for memory reads during consolidation
- `internal/store/meta.go` — Reference for `meta.json` format (do NOT import; re-parse JSON independently)

### Dependent Files
- `internal/daemon/daemon.go` (task_04) — Will create Service and wire SessionSpawner callback + periodic ticker

### Related ADRs
- [ADR-003: Frozen Snapshot Memory Injection With Dream-Only Extraction](adrs/adr-003.md) — Dream is the sole extraction mechanism

## Deliverables
- `internal/memory/lock.go` with ConsolidationLock implementation
- `internal/memory/dream.go` with Service, gates, and Run orchestration
- `internal/memory/prompt.go` with embedded consolidation prompt
- Unit tests with 80%+ coverage **(REQUIRED)**
- All tests pass with `-race` flag **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `TryAcquire` on non-existent lock file succeeds and creates file with PID
  - [x] `TryAcquire` on lock held by live PID returns `ok=false`
  - [x] `TryAcquire` on lock held by dead PID reclaims lock
  - [x] `TryAcquire` on lock older than stale age (1 hour) reclaims lock
  - [x] `Release` clears PID body and updates mtime to now
  - [x] `Rollback` clears PID body and restores prior mtime
  - [x] `LastConsolidatedAt` returns lock file mtime
  - [x] `LastConsolidatedAt` returns zero time when lock file doesn't exist
  - [x] `ShouldRun` returns false when time gate fails (< minHours since last)
  - [x] `ShouldRun` returns false when session gate fails (< minSessions since last)
  - [x] `ShouldRun` returns false when lock gate fails (already acquired)
  - [x] `ShouldRun` returns true when all three gates pass
  - [x] Gate evaluation order: time checked first, then sessions, then lock
  - [x] `Run` calls spawner with goal and prompt when gates pass
  - [x] `Run` calls `Release` on successful spawn
  - [x] `Run` calls `Rollback(priorMtime)` on failed spawn
  - [x] `Run` returns combined error (spawn + rollback) via `errors.Join`
  - [x] Session counting scans meta.json files and filters by StoppedAt >= since
  - [x] Session counting ignores sessions without StoppedAt (incomplete)
  - [x] Concurrent `Run()` calls are serialized via mutex
  - [x] Functional options override defaults correctly
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/memory/` package compiles with zero imports of session/, daemon/
- `make lint` passes with zero warnings

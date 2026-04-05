---
status: pending
title: Kernel & Session Integration
type: ""
complexity: high
dependencies:
    - task_01
    - task_02
    - task_03
---

# Task 04: Kernel & Session Integration

## Overview

Wire the memdir store and dream consolidation service into the kernel boot sequence and session lifecycle. This task connects the foundational packages (task_01, task_02, task_03) to the running system: initializing memory directories at boot, populating `MemoryContext` during prompt assembly, triggering dream consolidation on session stop and via a periodic daemon ticker, and exposing memory operations through HTTP API endpoints.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `MemoryDir` field to `config.HomePaths` struct
- MUST add `MemoryDir` to `ResolveHomePathsFrom()` path resolution
- MUST add `MemoryDir` to `EnsureHomeLayout()` directory creation
- MUST initialize `memdir.Store` in session creation with global (`HomePaths.MemoryDir`) and workspace (`<workspace>/.agh/memory/`) paths
- MUST call `memdir.Store.EnsureDirs()` during session initialization
- MUST build `MemoryContext` from `memdir.Store.LoadIndex()` (both scopes) + blackboard `type="memory"` query when assembling agent prompts
- MUST initialize `DreamService` during kernel boot (after SessionManager)
- MUST add a `time.Ticker` goroutine to the kernel lifecycle that checks `DreamService.ShouldRun()` at a configurable interval (default 30 minutes)
- MUST call `DreamService.ShouldRun()` after session stop and spawn consolidation if gates pass
- MUST implement `SessionSpawner` callback that creates an ephemeral session with a single worker agent for consolidation
- MUST register HTTP API routes: GET `/api/memory`, GET `/api/memory/:filename`, PUT `/api/memory/:filename`, DELETE `/api/memory/:filename`, POST `/api/memory/consolidate`
- MUST check available session slots before spawning consolidation session
- All existing kernel and session manager tests MUST continue to pass
</requirements>

## Subtasks

- [x] 4.1 Add `MemoryDir` to `HomePaths` struct and path resolution/directory creation
- [x] 4.2 Initialize `memdir.Store` in session creation alongside SQLite store
- [x] 4.3 Build `MemoryContext` during prompt assembly for agent spawning
- [x] 4.4 Initialize `DreamService` at kernel boot with functional options
- [x] 4.5 Add `time.Ticker` goroutine for periodic dream check in kernel lifecycle
- [x] 4.6 Hook dream trigger after session stop in `SessionManager.Stop()`
- [x] 4.7 Implement `SessionSpawner` callback for ephemeral consolidation sessions
- [x] 4.8 Register memory HTTP API endpoints (list, read, write, delete, consolidate)
- [x] 4.9 Write integration tests for all integration points

## Implementation Details

Modify existing files:

- `internal/config/config.go` — Add `MemoryDir` to `HomePaths` struct
- `internal/config/home.go` — Add `MemoryDir` to `ResolveHomePathsFrom()` and `EnsureHomeLayout()`
- `internal/kernel/kernel.go` — Initialize `DreamService` in `NewKernel()` boot sequence, add ticker goroutine in `runLifecycle()`
- `internal/kernel/session_manager.go` — Initialize `memdir.Store` in `Create()`/`initializeSession()`, build `MemoryContext` in prompt assembly, trigger dream in `Stop()`
- `internal/kernel/api.go` — Register memory HTTP endpoints in `registerKernelRoutes()`
- `internal/kernel/types.go` — Add `MemoryStore` field to `Session` struct, add `DreamService` field to `Kernel` struct

### Relevant Files

- `internal/kernel/kernel.go:197-416` — `NewKernel()` boot sequence (DreamService init goes here)
- `internal/kernel/kernel.go:565-608` — `runLifecycle()` (ticker goroutine goes here)
- `internal/kernel/session_manager.go:40-130` — `Create()` method (memdir.Store init goes here)
- `internal/kernel/session_manager.go:138-164` — `Stop()` method (dream trigger goes here)
- `internal/kernel/session_manager.go:385-464` — `bootstrapAgent()` (MemoryContext built here for prompt assembly)
- `internal/kernel/api.go:718-1216` — `registerKernelRoutes()` (memory routes go here)
- `internal/config/config.go:242-257` — `HomePaths` struct
- `internal/config/home.go:48-94` — `ResolveHomePathsFrom()` and `EnsureHomeLayout()`
- `internal/state/queries.go` — `ListBlackboard()` with type filter for team memory query

### Dependent Files

- `internal/cli/memory.go` (task_05) — CLI commands will call these HTTP API endpoints
- `internal/config/config_test.go` — May need updates for new `MemoryDir` field

### Related ADRs

- [ADR-001: Dual Storage](../adrs/adr-001.md) — Global + workspace directory paths
- [ADR-002: Dream via Ephemeral Session](../adrs/adr-002.md) — SessionSpawner callback pattern
- [ADR-004: time.Ticker Scheduling](../adrs/adr-004.md) — Periodic ticker in kernel lifecycle

## Deliverables

- `MemoryDir` integrated into `HomePaths` and home layout initialization
- `memdir.Store` initialized per-session with correct scope directories
- `MemoryContext` populated and passed to prompt assembler during agent spawning
- `DreamService` initialized at kernel boot with periodic ticker
- Dream triggered on session stop with session slot check
- 5 HTTP API endpoints for memory operations
- All existing tests still passing
- New integration tests with 80%+ coverage **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `HomePaths.MemoryDir` resolves to `<home>/memory`
  - [x] `EnsureHomeLayout()` creates memory directory
  - [x] Session creation initializes `memdir.Store` with correct global and workspace paths
  - [x] `MemoryContext` is populated in prompt assembly when memories exist
  - [x] `MemoryContext` is empty when no memories exist (backward compatible)
  - [x] Team memory query filters blackboard entries by `type="memory"`
  - [x] Dream ticker goroutine starts and stops with context cancellation
  - [x] Dream trigger on session stop calls `ShouldRun()`
  - [x] Consolidation session not spawned when at max session limit
- Integration tests:
  - [x] Boot test kernel with DreamService initialized, verify no panics
  - [x] Create session, verify memdir.Store is initialized and directories exist
  - [x] Spawn agent with memory files present, verify prompt contains memory index
  - [x] API: GET `/api/memory` returns memory headers
  - [x] API: PUT `/api/memory/:filename` creates memory file
  - [x] API: GET `/api/memory/:filename` returns file content
  - [x] API: DELETE `/api/memory/:filename` removes file
  - [x] API: POST `/api/memory/consolidate` triggers consolidation check
- Test coverage target: >=80%
- All tests must pass with `-race` flag
- Coverage evidence: touched task_04 production files reached `80.01% (2069/2586)` from `go test -coverprofile=/tmp/task04-all-cover.out ./internal/config ./internal/state ./internal/kernel/...`.

## Success Criteria

- All tests passing (including all pre-existing kernel, session, and config tests)
- Test coverage >=80%
- `make verify` passes
- Memory directories created on first boot
- Agent prompts include memory indexes when memories exist
- Dream consolidation triggers on session stop and periodic ticker
- HTTP API endpoints functional for all CRUD operations

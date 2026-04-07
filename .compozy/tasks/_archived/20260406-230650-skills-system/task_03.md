---
status: completed
title: Dual-scope registry with ForWorkspace
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Dual-scope registry with ForWorkspace

## Overview

Implement the thread-safe dual-scope skill registry that manages global skills (loaded at boot) and lazily merges workspace-scoped skills per session. This is the core data structure of the skills system — it follows the `memory.Store.ForWorkspace()` pattern already established in the codebase.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/registry.go` with `Registry` struct
- MUST implement `LoadAll(ctx)` to load global skills (bundled + user dirs) at boot
- MUST implement `ForWorkspace(ctx, workspace)` to lazily merge global + workspace skills
- MUST use `sync.RWMutex` for thread-safe concurrent access
- MUST use `atomic.Int64` for global version counter
- MUST cache workspace results in `wsCache` with mtime+size staleness check
- MUST evict workspace cache entries older than 10 minutes via `lastAccess` timestamp
- MUST call `VerifyContent()` on all non-bundled skills during loading
- MUST skip skills blocked by Critical severity warnings
- MUST implement override precedence: workspace > agents > user > bundled
- MUST log warnings on name collisions with override source info
- MUST implement `RefreshGlobal(ctx)` for watcher-triggered refresh
- MUST implement `Get(name)` and `List()` for global-only lookups
</requirements>

## Subtasks
- [x] 3.1 Implement `Registry` struct with global skills map, workspace cache, and version counter
- [x] 3.2 Implement `LoadAll()` that scans bundled FS + user directories with override precedence
- [x] 3.3 Implement `ForWorkspace()` with lazy loading and mtime-based cache invalidation
- [x] 3.4 Implement workspace cache TTL eviction (10 min)
- [x] 3.5 Implement `RefreshGlobal()` for atomic swap of global skills
- [x] 3.6 Write unit tests covering all registry operations and concurrency

## Implementation Details

See TechSpec "Dual-Scope Registry Design" section for the full design and "Loading Hierarchy" for precedence rules. Follow the `memory.Store.ForWorkspace()` pattern in `internal/memory/store.go`.

### Relevant Files
- `internal/memory/store.go` — ForWorkspace pattern to follow
- `internal/skills/types.go` — Domain types (task_01)
- `internal/skills/loader.go` — ParseSkillFile, scanDirectory (task_01)
- `internal/skills/verify.go` — VerifyContent (task_02)

### Dependent Files
- `internal/skills/catalog.go` — Will consume Registry via ForWorkspace (task_04)
- `internal/skills/watcher.go` — Will call RefreshGlobal (task_06)
- `daemon/daemon.go` — Will create Registry at boot (task_10)
- `cli/skill.go` — CLI will create ephemeral Registry (task_11)

### Related ADRs
- [ADR-002: Dual-Scope Registry](../adrs/adr-002.md) — Core design decision for global + workspace layers

## Deliverables
- `internal/skills/registry.go` with Registry implementation
- `internal/skills/registry_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] LoadAll loads bundled skills from fs.FS
  - [x] LoadAll loads user-level skills from filesystem directories
  - [x] User skill overrides bundled skill with same name
  - [x] ForWorkspace merges global + workspace skills correctly
  - [x] Workspace skill overrides global skill with same name
  - [x] ForWorkspace returns cached result when mtime unchanged
  - [x] ForWorkspace re-scans when mtime changed (cache invalidation)
  - [x] ForWorkspace returns different results for different workspaces
  - [x] Workspace cache entries evicted after 10 min of no access
  - [x] VerifyContent blocks Critical-severity skills during loading
  - [x] GlobalVersion increments on RefreshGlobal with actual changes
  - [x] GlobalVersion does NOT increment when no changes detected
  - [x] Concurrent Get/List calls under RWMutex do not deadlock
  - [x] Override collision logged with source info
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Concurrent read access is safe under load
- Different workspaces produce different skill sets

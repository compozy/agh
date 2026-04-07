---
status: completed
title: Hot-reload watcher
type: backend
complexity: medium
dependencies:
  - task_03
---

# Task 06: Hot-reload watcher

## Overview

Implement the stat-based polling watcher that detects changes to global skill directories and refreshes the registry without daemon restart. The watcher scans `~/.agh/skills/` and `~/.agents/skills/` for file changes using mtime+size comparisons, triggering atomic registry updates when changes are detected.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/watcher.go` with `Watcher` struct
- MUST poll global skill directories at configurable interval (default 3s)
- MUST use `os.Stat()` to collect mtime + size for each SKILL.md
- MUST compare against previous snapshot to detect additions, modifications, and deletions
- MUST call `registry.RefreshGlobal()` only when actual changes are detected
- MUST NOT poll bundled skills (immutable) or workspace directories (lazily checked)
- MUST run as a goroutine with explicit ownership via `context.Context` cancellation
- MUST use `select` with `ctx.Done()` for clean shutdown
- MUST NOT use `time.Sleep()` — use `time.NewTicker`
</requirements>

## Subtasks
- [x] 6.1 Implement `Watcher` struct with registry reference, interval, and root directories
- [x] 6.2 Implement `Start(ctx)` polling loop with ticker and context cancellation
- [x] 6.3 Implement change detection via mtime+size snapshot comparison
- [x] 6.4 Implement `detectChanges()` that scans roots and returns whether anything changed
- [x] 6.5 Write unit tests for change detection and lifecycle management

## Implementation Details

See TechSpec "Hot-Reload (F10)" section and ADR-004. The watcher only polls global directories — workspace directories are checked lazily in `ForWorkspace()`.

Follow AGH concurrency conventions: goroutine tracked with `sync.WaitGroup` or equivalent, `select` with `ctx.Done()`, no fire-and-forget goroutines.

### Relevant Files
- `internal/skills/registry.go` — RefreshGlobal() called on change detection (task_03)
- `internal/daemon/daemon.go` — Dream consolidation loop as goroutine lifecycle pattern reference

### Dependent Files
- `daemon/daemon.go` — Will start/stop Watcher in boot/shutdown (task_10)

### Related ADRs
- [ADR-004: Stat-Based Polling for Hot-Reload](../adrs/adr-004.md) — Polling over fsnotify for reliability

## Deliverables
- `internal/skills/watcher.go` with Watcher implementation
- `internal/skills/watcher_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Detect new SKILL.md added to watched directory
  - [x] Detect modified SKILL.md (mtime change)
  - [x] Detect deleted SKILL.md
  - [x] No false positive when mtime unchanged
  - [x] Context cancellation stops the polling loop cleanly
  - [x] Watcher does not poll bundled or workspace directories
  - [x] Multiple polling cycles with no changes do not trigger refresh
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Watcher shuts down cleanly on context cancellation
- No goroutine leaks

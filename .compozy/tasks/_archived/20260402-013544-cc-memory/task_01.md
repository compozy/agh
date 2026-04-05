---
status: completed
title: Memdir Core Package
type: ""
complexity: medium
dependencies: []
---

# Task 01: Memdir Core Package

## Overview

Implement the foundational `internal/kernel/memdir/` package that provides file-based persistent memory storage with YAML frontmatter metadata, dual-scope directory management (global + workspace), MEMORY.md index loading with truncation, and mtime-based staleness tracking. This package is the foundation all other memory tasks depend on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST define closed 4-type taxonomy: `user`, `feedback`, `project`, `reference`
- MUST define `Scope` type with `global` and `workspace` values
- MUST implement `Store` struct with `globalDir` and `workspaceDir` fields
- MUST implement `Scan()` returning `[]MemoryHeader` sorted newest-first, capped at 200 files
- MUST implement `Read()` returning raw file bytes for a given scope + filename
- MUST implement `Write()` persisting content to the correct scope directory
- MUST implement `Delete()` removing a file and its index entry
- MUST implement `LoadIndex()` returning MEMORY.md content with truncation at 200 lines / 25KB
- MUST implement `EnsureDirs()` creating both scope directories if missing
- MUST reuse existing `internal/frontmatter/` package for YAML parsing
- MUST implement staleness functions: `AgeDays()`, `AgeText()`, `FreshnessWarning()`
- `FreshnessWarning()` MUST return empty string for memories ≤ 1 day old and a staleness caveat for older memories
</requirements>

## Subtasks

- [x] 1.1 Define types: `MemoryType`, `MemoryHeader`, `Scope`, `Store` struct
- [x] 1.2 Implement `NewStore()`, `EnsureDirs()`, and scope directory resolution
- [x] 1.3 Implement `Write()` with frontmatter validation and `Read()` with file content return
- [x] 1.4 Implement `Delete()` with index entry removal
- [x] 1.5 Implement `Scan()` with frontmatter parsing, newest-first sorting, and 200-file cap
- [x] 1.6 Implement `LoadIndex()` with 200-line / 25KB truncation and `wasTruncated` flag
- [x] 1.7 Implement staleness functions in `staleness.go`
- [x] 1.8 Write comprehensive unit tests for all operations

## Implementation Details

Create new package at `internal/kernel/memdir/` with the following files:

- `memdir.go` — `Store` struct, `NewStore()`, `EnsureDirs()`, `Write()`, `Read()`, `Delete()`, `Scan()`, `LoadIndex()`
- `types.go` — `MemoryType` constants, `MemoryHeader`, `Scope` constants
- `staleness.go` — `AgeDays()`, `AgeText()`, `FreshnessWarning()`
- `memdir_test.go` — unit tests for store operations
- `staleness_test.go` — unit tests for staleness functions

### Relevant Files

- `internal/frontmatter/frontmatter.go` — Reuse `Parse()` and `Format()` for YAML frontmatter handling
- `internal/config/config.go:242-257` — `HomePaths` struct (reference for directory path patterns)
- `internal/config/home.go:48-94` — `ResolveHomePathsFrom()` and `EnsureHomeLayout()` (reference for directory creation patterns)

### Dependent Files

- `internal/kernel/dream/` (task_02) — depends on `memdir.Store` for memory access
- `internal/prompt/assembler.go` (task_03) — depends on `memdir.Store.LoadIndex()` for prompt injection
- `internal/kernel/session_manager.go` (task_04) — depends on `memdir.Store` initialization

### Related ADRs

- [ADR-001: Dual Storage](../adrs/adr-001.md) — Defines the global + workspace dual-directory architecture
- [ADR-005: Index-Only Prompt Injection](../adrs/adr-005.md) — Defines MEMORY.md index format and truncation limits

## Deliverables

- `internal/kernel/memdir/` package with types, store, and staleness modules
- All store operations working for both global and workspace scopes
- MEMORY.md index loading with truncation
- Staleness functions with day-boundary accuracy
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Write and Read round-trip for global scope
  - [x] Write and Read round-trip for workspace scope
  - [x] Write with invalid frontmatter (missing name, missing type) returns error
  - [x] Read non-existent file returns descriptive error
  - [x] Delete removes file and returns no error
  - [x] Delete non-existent file returns descriptive error
  - [x] Scan returns headers sorted newest-first
  - [x] Scan caps results at 200 files
  - [x] Scan handles malformed frontmatter files gracefully (skips with warning)
  - [x] Scan on empty directory returns empty slice
  - [x] LoadIndex returns full content when under 200 lines / 25KB
  - [x] LoadIndex truncates at 200 lines and sets `wasTruncated` flag
  - [x] LoadIndex truncates at 25KB and sets `wasTruncated` flag
  - [x] LoadIndex on missing MEMORY.md returns empty string, no error
  - [x] EnsureDirs creates both directories when missing
  - [x] EnsureDirs is idempotent on existing directories
  - [x] AgeDays returns 0 for today, 1 for yesterday
  - [x] AgeText returns "today", "yesterday", "N days ago"
  - [x] FreshnessWarning returns empty for ≤ 1 day old memories
  - [x] FreshnessWarning returns staleness caveat for > 1 day old memories
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- Package has zero external dependencies beyond `internal/frontmatter/`
- All 4 memory types and 2 scopes fully functional

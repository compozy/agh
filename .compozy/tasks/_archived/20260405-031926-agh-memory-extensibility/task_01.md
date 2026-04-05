---
status: completed
title: Memory Store Core (memdir)
type: ""
complexity: medium
dependencies: []
---

# Task 01: Memory Store Core (memdir)

## Overview

Implement the file-based persistent memory store as the `internal/memory/` package. This is the foundation for all memory features — it provides CRUD operations on memory files with YAML frontmatter metadata, dual-scope directories (global + workspace), MEMORY.md index management, staleness tracking, and filename/frontmatter validation. Ported from the proven cc-memory implementation, adapted to AGH v2 conventions.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Implementation Design > Core Interfaces" and "Data Models" sections for Store API and types
- REFERENCE `.old_project/internal/kernel/memdir/` for the proven implementation to port
- REFERENCE `.resources/claude-code/memdir/` for Claude Code's memory file format and index patterns
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/memory/` package with zero imports of `session/`, `daemon/`, `httpapi/`, `udsapi/`, or `cli/`
- MUST implement `Store` struct with `NewStore(globalDir string) *Store` constructor
- MUST implement CRUD operations: `Read`, `Write`, `Delete`, `Scan`, `LoadIndex`, `EnsureDirs`
- MUST support dual scope: `ScopeGlobal` and `ScopeWorkspace` with directory resolution per scope
- MUST parse YAML frontmatter using `github.com/goccy/go-yaml` (already in go.mod)
- MUST validate frontmatter: require `name` and `type` fields, validate type against closed 4-type taxonomy
- MUST support `agent_name` optional field in frontmatter for team memory tracking
- MUST cap MEMORY.md index at 200 lines / 25,000 bytes with UTF-8 boundary-safe truncation
- MUST cap `Scan` results at 200 files, sorted newest-first by ModTime
- MUST implement staleness functions: `AgeDays`, `AgeText`, `FreshnessWarning` with 1-day threshold
- MUST use atomic writes (write-to-temp + rename) for file persistence
- MUST reject filenames containing path separators or equal to "." or ".."
- MUST follow error wrapping pattern: `fmt.Errorf("memory: <operation> %q: %w", path, err)`
</requirements>

## Subtasks

- [x] 1.1 Define memory types (`MemoryType`, `Scope`, `MemoryHeader`) and validation logic
- [x] 1.2 Implement `Store` struct with constructor and directory resolution per scope
- [x] 1.3 Implement `EnsureDirs`, `Write`, `Read`, `Delete` operations with atomic writes
- [x] 1.4 Implement `Scan` (newest-first, 200-file cap) with frontmatter parsing
- [x] 1.5 Implement `LoadIndex` with line/byte truncation and UTF-8 boundary handling
- [x] 1.6 Implement `removeIndexEntry` for cleaning MEMORY.md on file delete
- [x] 1.7 Implement staleness functions (`AgeDays`, `AgeText`, `FreshnessWarning`)

## Implementation Details

Create the `internal/memory/` package with these files:

- `types.go` — `MemoryType`, `Scope`, `MemoryHeader` types, validation, constants
- `store.go` — `Store` struct, CRUD operations, index management
- `staleness.go` — Age calculation and freshness warning functions

Reference the TechSpec "Data Models" section for exact type definitions. Reference TechSpec "Filesystem Layout" for directory structure.

### Relevant Files

- `.old_project/internal/kernel/memdir/memdir.go` — Proven Store implementation to port
- `.old_project/internal/kernel/memdir/types.go` — Type definitions and validation patterns
- `.old_project/internal/kernel/memdir/staleness.go` — Staleness calculation logic
- `.old_project/internal/kernel/memdir/memdir_test.go` — Test patterns and edge cases
- `.resources/claude-code/memdir/` — Claude Code's memory file format for reference
- `.resources/hermes/plugins/memory/` — Hermes memory tool patterns for comparison
- `internal/config/config.go` — AGH v2 coding conventions and struct patterns

### Dependent Files

- `internal/memory/dream.go` (task_03) — Will use `Store` for memory reads during consolidation
- `internal/memory/assembler.go` (task_04) — Will use `Store.LoadIndex()` for prompt assembly
- `internal/httpapi/memory.go` (task_05) — Will use `Store` for API handlers
- `internal/cli/memory.go` (task_05) — Will use `Store` via API for CLI commands

### Related ADRs

- [ADR-004: Four-Type Memory Taxonomy](adrs/adr-004.md) — Constrains the type enum to 4 values (user, feedback, project, reference)

## Deliverables

- `internal/memory/types.go` with all type definitions, constants, and validation
- `internal/memory/store.go` with full CRUD + index operations
- `internal/memory/staleness.go` with age tracking functions
- Unit tests with 80%+ coverage **(REQUIRED)**
- All tests pass with `-race` flag **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `Write` with valid frontmatter persists file and content matches on `Read`
  - [x] `Write` with missing `name` field returns validation error
  - [x] `Write` with unknown `type` value returns validation error
  - [x] `Write` with path separator in filename returns error
  - [x] `Write` with "." or ".." as filename returns error
  - [x] `Write` to global scope persists in globalDir, workspace scope in workspaceDir
  - [x] `Read` of non-existent file returns `os.ErrNotExist`
  - [x] `Delete` removes file from disk
  - [x] `Delete` removes entry from MEMORY.md index
  - [x] `Scan` returns headers sorted newest-first
  - [x] `Scan` caps at 200 entries
  - [x] `Scan` skips files with malformed frontmatter and logs warning
  - [x] `LoadIndex` returns MEMORY.md content
  - [x] `LoadIndex` truncates at 200 lines, returns `truncated=true`
  - [x] `LoadIndex` truncates at 25,000 bytes, respects UTF-8 boundaries
  - [x] `LoadIndex` returns empty string when MEMORY.md doesn't exist
  - [x] `EnsureDirs` creates both global and workspace directories
  - [x] `AgeDays` returns 0 for today, 1 for yesterday
  - [x] `AgeText` returns "today", "yesterday", "N days ago"
  - [x] `FreshnessWarning` returns empty string for ≤1 day, caveat for >1 day
  - [x] Frontmatter `agent_name` field is parsed when present, ignored when absent
  - [x] `MemoryType` validation normalizes input (trim, lowercase)
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `internal/memory/` package compiles with zero imports of session/, daemon/, httpapi/, udsapi/, cli/
- `make lint` passes with zero warnings

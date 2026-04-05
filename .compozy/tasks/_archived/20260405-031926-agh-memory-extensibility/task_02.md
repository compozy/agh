---
status: completed
title: Config, Session & Store Scaffolding
type: ""
complexity: medium
dependencies: []
---

# Task 02: Config, Session & Store Scaffolding

## Overview

Add the foundational scaffolding across three existing packages to support the memory system: (1) `MemoryConfig` and `DreamConfig` types in config/ with HomePaths.MemoryDir; (2) `PromptAssembler` interface and `SessionType` enum in session/; (3) `session_type` column in the global store schema. These are all additive changes with no behavioral modifications to existing code — they prepare the seams that tasks 03-05 will wire into.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Data Models" for config types and SessionType enum
- REFERENCE TECHSPEC "Core Interfaces" for PromptAssembler interface signature
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `MemoryConfig` struct with `Enabled bool`, `GlobalDir string`, `Dream DreamConfig` fields to `Config`
- MUST add `DreamConfig` struct with `Enabled bool`, `Agent string`, `MinHours float64`, `MinSessions int`, `CheckInterval` fields
- MUST add `MemoryDir` field to `HomePaths` struct with default `~/.agh/memory/`
- MUST add `MemoryDirName` constant and wire into `ResolveHomePaths()` and `EnsureHomeLayout()`
- MUST set sensible defaults in `DefaultWithHome()`: memory enabled, dream enabled, agent "claude", min_hours 24, min_sessions 3, check_interval 30m
- MUST add `PromptAssembler` interface to `session/interfaces.go` with single method: `Assemble(ctx, AgentDef, workspace) (string, error)`
- MUST add `SessionType` type and constants (`SessionTypeUser`, `SessionTypeDream`, `SessionTypeSystem`) to session/
- MUST add `Type SessionType` field to `CreateOpts` with default `SessionTypeUser`
- MUST add `WithPromptAssembler(PromptAssembler) Option` functional option to session Manager
- MUST add `session_type TEXT NOT NULL DEFAULT 'user'` column to global `sessions` table schema
- MUST update `SessionInfo` and `RegisterSession` in store to include session type
- MUST NOT change any existing behavior — all additions must be backward-compatible with defaults
</requirements>

## Subtasks
- [x] 2.1 Add `MemoryConfig`, `DreamConfig` structs to config with TOML tags and defaults
- [x] 2.2 Add `MemoryDir` to `HomePaths`, wire into `ResolveHomePaths()` and `EnsureHomeLayout()`
- [x] 2.3 Add `PromptAssembler` interface and `SessionType` enum to session package
- [x] 2.4 Add `Type` field to `CreateOpts`, `WithPromptAssembler` option to Manager
- [x] 2.5 Add `session_type` column to global sessions schema and update store types
- [x] 2.6 Update config validation and merge logic for new sections

## Implementation Details

Modify these existing files (additive only):
- `internal/config/config.go` — Add `MemoryConfig`, `DreamConfig` to `Config` struct, add defaults in `DefaultWithHome()`
- `internal/config/home.go` — Add `MemoryDir` to `HomePaths`, `MemoryDirName` constant, wire into resolution/layout
- `internal/session/interfaces.go` — Add `PromptAssembler` interface after existing `Notifier` interface
- `internal/session/session.go` — Add `SessionType` type and constants
- `internal/session/manager.go` — Add `Type` to `CreateOpts`, add `assembler` field to Manager, add `WithPromptAssembler` option
- `internal/store/schema.go` — Add `session_type` column to global sessions DDL
- `internal/store/store.go` — Add `SessionType` field to `SessionInfo`
- `internal/store/global_db.go` — Update `RegisterSession` and `ListSessions` for session_type

Reference TechSpec "Data Models > Configuration additions" for exact struct definitions.

### Relevant Files
- `internal/config/config.go:77-87` — Current Config struct to extend
- `internal/config/home.go:31-43` — Current HomePaths struct to extend
- `internal/config/config.go:174-209` — DefaultWithHome() where defaults are set
- `internal/session/interfaces.go:153-158` — Current Notifier interface (add PromptAssembler nearby)
- `internal/session/session.go:20-28` — Current SessionState type (add SessionType similarly)
- `internal/session/manager.go:41-46` — Current CreateOpts struct to extend
- `internal/session/manager.go:63-82` — Manager struct to add assembler field
- `internal/store/schema.go:49-57` — Current sessions table DDL
- `internal/store/store.go:142-151` — Current SessionInfo struct
- `internal/store/global_db.go` — RegisterSession and ListSessions methods

### Dependent Files
- `internal/memory/assembler.go` (task_04) — Will implement `PromptAssembler` interface
- `internal/daemon/daemon.go` (task_04) — Will use config types and inject assembler
- `internal/daemon/daemon.go` (task_04) — Will bridge `DreamConfig` values to dream Service functional options

### Related ADRs
- [ADR-002: PromptAssembler Interface in session/](adrs/adr-002.md) — Defines where the interface lives and why
- [ADR-003: Frozen Snapshot Memory Injection](adrs/adr-003.md) — Constrains when assembly happens (session start only)

## Deliverables
- Modified `internal/config/config.go` and `config/home.go` with memory config types and HomePaths
- Modified `internal/session/interfaces.go`, `session.go`, `manager.go` with interface and type additions
- Modified `internal/store/schema.go`, `store.go`, `global_db.go` with session_type column
- Unit tests with 80%+ coverage for new code paths **(REQUIRED)**
- All existing tests continue to pass **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Config loading with `[memory]` TOML section parses `MemoryConfig` correctly
  - [x] Config loading without `[memory]` section uses defaults (enabled, dream.agent="claude", etc.)
  - [x] `DreamConfig` validates `min_hours > 0` and `min_sessions > 0`
  - [x] `HomePaths.MemoryDir` defaults to `~/.agh/memory/`
  - [x] `EnsureHomeLayout()` creates memory directory
  - [x] `CreateOpts` with empty `Type` defaults to `SessionTypeUser`
  - [x] `WithPromptAssembler(nil)` is safe (no panic)
  - [x] Manager with nil assembler skips assembly in Create()
  - [x] `SessionInfo.SessionType` persists and reads correctly from global DB
  - [x] `RegisterSession` with session_type writes to DB
  - [x] `ListSessions` returns session_type field
  - [x] All existing session, config, and store tests continue to pass
- Test coverage target: >=80% on new code
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing (new + existing)
- Test coverage >=80% on new code paths
- `make lint` passes with zero warnings
- `make verify` passes
- No behavioral changes to existing functionality

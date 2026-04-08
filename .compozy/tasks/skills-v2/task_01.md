---
status: completed
title: "Extend types with MCPServerDecl, HookDecl, Provenance, and SourceMarketplace"
type: backend
complexity: low
dependencies: []
---

# Task 1: Extend types with MCPServerDecl, HookDecl, Provenance, and SourceMarketplace

## Overview

Add the foundational type definitions that all other skills-v2 tasks depend on. This includes MCP server declarations, lifecycle hook declarations, marketplace provenance metadata, and the new `SourceMarketplace` skill source level. Every subsequent task imports these types.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `MCPServerDecl` struct with Name, Command, Args, Env fields to `internal/skills/types.go`
- MUST add `HookDecl` struct with Event, Command, Args, Timeout, Env fields
- MUST add `HookEvent` type with `HookSessionCreated` and `HookSessionStopped` constants
- MUST add `Provenance` struct with Hash, Registry, Slug, Version, InstalledAt fields (JSON tags)
- MUST insert `SourceMarketplace` between `SourceBundled` and `SourceUser` in the `SkillSource` iota block
- MUST add `MCPServers`, `Hooks`, `Provenance`, `InstalledFrom` fields to the existing `Skill` struct
- MUST update `skillSourceName()` to handle the new `SourceMarketplace` case
- MUST update `skillSourceFromWorkspacePath()` if needed for marketplace source
- MUST ensure `cloneSkill()` deep-copies the new fields
</requirements>

## Subtasks
- [x] 1.1 Add MCPServerDecl, HookDecl, HookEvent, and Provenance types to types.go
- [x] 1.2 Insert SourceMarketplace into SkillSource iota and update helper functions
- [x] 1.3 Extend Skill struct with MCPServers, Hooks, Provenance, InstalledFrom fields
- [x] 1.4 Update cloneSkill/cloneSkillMeta to deep-copy new fields
- [x] 1.5 Write unit tests for all new types and clone behavior

## Implementation Details

All changes are in `internal/skills/types.go` and the clone helpers in `internal/skills/registry.go`.

See TechSpec "Core Interfaces" and "Data Models" sections for exact type definitions.

### Relevant Files
- `internal/skills/types.go` — SkillSource const block (lines 27-39), Skill struct (lines 17-25), SkillMeta (lines 9-15)
- `internal/skills/registry.go` — cloneSkill (line 475), cloneSkillMeta (line 486), skillSourceName (line 619), skillSourceFromWorkspacePath (line 634)

### Dependent Files
- `internal/skills/loader.go` — will need to populate new Skill fields (task_03)
- `internal/skills/catalog.go` — may need to handle SourceMarketplace in display
- All task_02 through task_10 depend on these types

### Related ADRs
- [ADR-001: MCP Consent Model](adrs/adr-001.md) — defines the trust tiers that SourceMarketplace enables
- [ADR-004: Hash-Based Provenance](adrs/adr-004.md) — Provenance struct stores hash metadata

## Deliverables
- Extended `types.go` with all new types
- Updated clone helpers in `registry.go`
- Updated `skillSourceName()` for marketplace
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] MCPServerDecl can be instantiated with all fields populated
  - [x] HookDecl with valid HookEvent constants (HookSessionCreated, HookSessionStopped)
  - [x] HookEvent string values match "on_session_created" and "on_session_stopped"
  - [x] SourceMarketplace iota value is between SourceBundled and SourceUser
  - [x] skillSourceName returns "marketplace" for SourceMarketplace
  - [x] cloneSkill deep-copies MCPServers slice (mutation isolation)
  - [x] cloneSkill deep-copies Hooks slice (mutation isolation)
  - [x] cloneSkill deep-copies Provenance pointer (nil and non-nil cases)
  - [x] cloneSkill preserves InstalledFrom string
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- All 5 new types compile and are importable by dependent packages

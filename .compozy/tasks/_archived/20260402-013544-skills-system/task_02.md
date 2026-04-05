---
status: completed
title: Registry, Catalog Builder and Bundled Skills
type: ""
complexity: medium
dependencies:
    - task_01
---

# Task 2: Registry, Catalog Builder and Bundled Skills

## Overview

Create the skill registry (thread-safe in-memory store with 4-level loading pipeline), the XML catalog builder for system prompt injection, and the bundled skills infrastructure via `go:embed`. The registry orchestrates discovery and loading from all skill directories, the catalog formats the `<available_skills>` XML block, and bundled skills provide out-of-the-box defaults.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement thread-safe Registry with sync.RWMutex following existing registry patterns in internal/registry/
- MUST load skills in precedence order: bundled → user (~/.agh/skills/ + ~/.agents/skills/) → .agents/skills/ → .agh/skills/
- MUST override same-name skills from higher-precedence sources and log warnings on collisions
- MUST generate XML catalog in AgentSkills format with `<available_skills>` block
- MUST include behavioral instructions in catalog telling agents to use `agh skill view <name>`
- MUST embed at least one example bundled skill via go:embed
- MUST support Freeze() to prevent post-boot loading
- MUST generate SkillSnapshot with version tracking
</requirements>

## Subtasks
- [x] 2.1 Create `internal/skills/registry.go` with Registry struct (NewRegistry, LoadAll, LoadDir, Get, List, Snapshot, Freeze)
- [x] 2.2 Implement 4-level loading pipeline in LoadAll() with precedence override semantics
- [x] 2.3 Create `internal/skills/catalog.go` with BuildCatalog() that generates XML catalog string with behavioral instructions
- [x] 2.4 Create `internal/skills/bundled/embed.go` with go:embed directive and at least one example skill
- [x] 2.5 Create `internal/skills/bundled/skills/` directory with example SKILL.md files
- [x] 2.6 Wire bundled FS loading into Registry.LoadAll()

## Implementation Details

### Relevant Files
- `internal/skills/registry.go` — New file: registry implementation
- `internal/skills/catalog.go` — New file: XML catalog builder
- `internal/skills/bundled/embed.go` — New file: go:embed directive
- `internal/skills/bundled/skills/` — New directory: embedded skill files
- `internal/registry/agents.go` — Reference: existing registry pattern (sync.RWMutex, map, List/Lookup methods)
- `internal/registry/drivers.go` — Reference: simpler registry pattern

### Dependent Files
- `internal/kernel/kernel.go` (task_03) — will instantiate Registry at boot
- `internal/kernel/session_manager.go` (task_03) — will call Snapshot() during agent spawn
- `internal/cli/skill.go` (task_04) — will use Registry for list/view/info commands

### Related ADRs
- [ADR-004: Four-Level Loading Hierarchy](adrs/adr-004.md) — Defines the precedence order this registry must implement
- [ADR-003: System Prompt + CLI Access](adrs/adr-003.md) — Defines the catalog format and behavioral instructions

## Deliverables
- `internal/skills/registry.go` with full Registry implementation
- `internal/skills/catalog.go` with BuildCatalog()
- `internal/skills/bundled/embed.go` with embedded skills
- At least one bundled skill SKILL.md
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] NewRegistry returns empty registry
  - [x] LoadDir loads all valid skills from directory
  - [x] LoadDir skips directories without SKILL.md
  - [x] LoadDir skips skills that fail verification (critical severity)
  - [x] LoadAll loads from multiple directories in correct precedence order
  - [x] Higher-precedence source overrides lower-precedence same-name skill
  - [x] Name collision logs warning via slog
  - [x] Get returns skill by name, returns false for missing
  - [x] List returns all skills sorted by name
  - [x] Snapshot filters by eligibility and returns immutable SkillSnapshot
  - [x] Snapshot version increments on registry changes
  - [x] Freeze prevents further LoadDir calls
  - [x] Concurrent read access is safe (multiple goroutines calling Get/List)
  - [x] BuildCatalog generates valid XML with `<available_skills>` wrapper
  - [x] BuildCatalog includes behavioral instructions mentioning `agh skill view`
  - [x] BuildCatalog escapes XML special characters in name/description
  - [x] BuildCatalog returns empty string for empty skill list
  - [x] Bundled skills load via embed.FS
- Integration tests:
  - [x] End-to-end: populate temp dirs with skills → LoadAll → Snapshot → verify catalog contains all expected skills
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Registry correctly loads and overrides skills from 4 levels
- BuildCatalog generates AgentSkills-compliant XML
- Bundled skills are discoverable after LoadAll
- `make verify` passes (fmt + lint + test + build)

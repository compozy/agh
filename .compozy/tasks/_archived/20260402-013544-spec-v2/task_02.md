---
status: completed
domain: Config
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
---

# Task 2: Configuration System

> **Note:** Multi-session config merge ($AGH_HOME, global+workspace merge) is handled by Task 08.

## Overview
Rewrite the configuration system to support the AGH-specific config.toml schema with all sections (limits, dashboard, runtime, meta, drivers), role loading from roles/*.toml files, playbook loading from playbooks/*.md files, and comprehensive validation.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST load config.toml with all sections defined in docs/spec-v2/07-configuration.md
- MUST provide sensible defaults for all config values per the spec defaults table
- MUST validate all config values (port ranges, duration parsing, driver names, etc.)
- MUST load role definitions from roles/*.toml and roles/*.draft.toml
- MUST load playbook files from playbooks/*.md and playbooks/*.draft.md
- MUST distinguish draft vs approved status for roles and playbooks
- MUST support the `agh init` filesystem layout creation (~/.agh/config.toml, roles/, playbooks/, sessions/)
- MUST parse duration strings (e.g., "5s", "30s", "60s") for backoff and interval configs
- MUST use BurntSushi/toml for TOML parsing (already in go.mod)
</requirements>

## Subtasks
- [x] 2.1 Implement config.toml loading with all sections and default values
- [x] 2.2 Implement config validation (port ranges, durations, required fields, driver existence)
- [x] 2.3 Implement role catalog loading from roles/*.toml with draft detection
- [x] 2.4 Implement playbook loading from playbooks/*.md with draft detection
- [x] 2.5 Implement filesystem layout creation for `agh init`
- [x] 2.6 Implement configuration resolution order (bootstrap → role → default)

## Implementation Details
Refer to docs/spec-v2/07-configuration.md for the complete config schema and resolution order. Refer to docs/spec-v2/03-agents.md for role definition format.

### Relevant Files
- `internal/config/config.go` — existing config, needs full rewrite
- `internal/config/env.go` — existing env loading, adapt or replace
- `docs/spec-v2/07-configuration.md` — complete config reference
- `docs/spec-v2/03-agents.md` — role definition format

### Dependent Files
- `internal/kernel/types.go` — Config, RoleConfig structs defined in task_01
- `cmd/agh/main.go` — will call config.Load()

## Deliverables
- Rewritten internal/config/ package with full config.toml support
- Role catalog loader (roles/*.toml)
- Playbook loader (playbooks/*.md)
- Filesystem layout initializer for `agh init`
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for config round-trip (write → load → validate) **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Load valid config.toml with all sections populated
  - [x] Default values applied when sections are missing
  - [x] Validation rejects invalid port (0, 65536+)
  - [x] Validation rejects invalid duration strings
  - [x] Validation rejects unknown driver names
  - [x] Role loading reads both .toml and .draft.toml with correct status
  - [x] Playbook loading reads both .md and .draft.md with correct status
  - [x] Driver resolution order: bootstrap → role → default
  - [x] Filesystem layout creation produces correct directory structure
- Integration tests:
  - [x] Config round-trip: create temp dir, write config, load, validate all fields match
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Config loads all sections from docs/spec-v2/07-configuration.md example
- Roles and playbooks load correctly with draft status detection

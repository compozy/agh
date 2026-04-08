---
status: completed
title: "Extend config with MarketplaceConfig and merge plumbing"
type: backend
complexity: medium
dependencies: []
---

# Task 2: Extend config with MarketplaceConfig and merge plumbing

## Overview

Add `AllowedMarketplaceMCP` and `MarketplaceConfig` fields to `SkillsConfig`, create the corresponding overlay struct for config merge, and update validation. This enables marketplace consent persistence and registry selection via TOML config.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `AllowedMarketplaceMCP []string` field to `SkillsConfig` in config.go
- MUST add `Marketplace MarketplaceConfig` nested struct to `SkillsConfig`
- MUST define `MarketplaceConfig` with `Registry` (default "clawhub") and `BaseURL` fields
- MUST create matching overlay types in `merge.go` (`marketplaceOverlay` with pointer fields)
- MUST extend `skillsOverlay` with `AllowedMarketplaceMCP` and `Marketplace` overlay fields
- MUST implement `Apply()` methods for the new overlay types
- MUST update `SkillsConfig.Validate()` to validate marketplace config when present
- MUST set sensible defaults in `DefaultWithHome()` (marketplace disabled/empty by default)
</requirements>

## Subtasks
- [ ] 2.1 Add MarketplaceConfig struct and extend SkillsConfig in config.go
- [ ] 2.2 Create marketplaceOverlay and extend skillsOverlay in merge.go
- [ ] 2.3 Implement Apply() methods for new overlay types
- [ ] 2.4 Update Validate() for marketplace config fields
- [ ] 2.5 Write unit tests for config parsing, merging, and validation

## Implementation Details

Changes span `internal/config/config.go` and `internal/config/merge.go`. Follow the existing overlay pointer pattern (see `skillsOverlay` at merge.go lines 88-92).

See TechSpec "Config extensions" section for field definitions.

### Relevant Files
- `internal/config/config.go` — SkillsConfig (lines 97-102), Validate (lines 427-437), DefaultWithHome (line 280)
- `internal/config/merge.go` — skillsOverlay (lines 88-92), Apply (lines 265-275)

### Dependent Files
- `internal/cli/skill.go` — marketplace CLI commands read config (task_10)
- `internal/skills/mcp.go` — MCPResolver reads AllowedMarketplaceMCP (task_04)
- `internal/daemon/boot.go` — passes config to MCPResolver (task_09)

### Related ADRs
- [ADR-001: MCP Consent Model](adrs/adr-001.md) — AllowedMarketplaceMCP stores consent list
- [ADR-003: Pluggable Registry Interface](adrs/adr-003.md) — MarketplaceConfig.Registry selects backend

## Deliverables
- Extended `SkillsConfig` with marketplace fields
- Overlay types and Apply() methods in merge.go
- Updated validation
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Parse TOML with `[skills.marketplace]` section into MarketplaceConfig
  - [ ] Parse `allowed_marketplace_mcp = ["skill-a", "skill-b"]` into SkillsConfig
  - [ ] Workspace overlay merges MarketplaceConfig.Registry over global default
  - [ ] Workspace overlay merges AllowedMarketplaceMCP (replaces, not appends)
  - [ ] Nil overlay fields leave defaults untouched
  - [ ] Validate() accepts valid marketplace config
  - [ ] Validate() rejects empty Registry string when Marketplace is explicitly configured
  - [ ] DefaultWithHome() sets empty AllowedMarketplaceMCP and no marketplace config
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Config round-trips through TOML parse → overlay merge → validation without error

---
status: completed
title: "Add marketplace CLI commands (search/install/remove/update)"
type: backend
complexity: medium
dependencies:
  - task_07
  - task_08
  - task_02
---

# Task 10: Add marketplace CLI commands (search/install/remove/update)

## Overview

Add four new subcommands to `agh skill`: `search`, `install`, `remove`, and `update`. These commands construct a marketplace client from config and dispatch through the `marketplace.Registry` interface. Install includes archive extraction, VerifyContent scanning, hash computation, and sidecar writing.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `search` subcommand: calls `Registry.Search()`, displays results in table format
- MUST add `install` subcommand: calls `Registry.Download()`, extracts to `~/.agh/skills/<name>/`, runs VerifyContent, computes hash, writes `.agh-meta.json` sidecar
- MUST add `remove` subcommand: deletes skill directory from `~/.agh/skills/` (only marketplace-installed skills with sidecar)
- MUST add `update` subcommand: checks for newer version, re-downloads if available, updates sidecar
- MUST construct marketplace client from `MarketplaceConfig` in config
- MUST block install if VerifyContent returns critical findings (with descriptive error message)
- MUST support `--limit` flag on search command
- MUST support `--all` flag on update command
- MUST register all new commands in `newSkillCommand()` function
</requirements>

## Subtasks
- [x] 10.1 Implement newSkillSearchCommand with Registry.Search() dispatch and table output
- [x] 10.2 Implement newSkillInstallCommand with download, extraction, verification, and sidecar
- [x] 10.3 Implement newSkillRemoveCommand with sidecar detection and directory cleanup
- [x] 10.4 Implement newSkillUpdateCommand with version check and re-install
- [x] 10.5 Register all new commands in newSkillCommand()
- [x] 10.6 Write unit tests for command argument parsing and integration tests for install flow

## Implementation Details

All changes in `internal/cli/skill.go`. Follow existing command patterns (see `newSkillListCommand` at line 90, `newSkillCreateCommand` at line 210). Marketplace client constructed per-command from config.

See TechSpec "CLI Commands" and "Data Flow — Marketplace" sections.

### Relevant Files
- `internal/cli/skill.go` — newSkillCommand (line 77), existing command patterns
- `internal/skills/marketplace/registry.go` — Registry interface (from task_08)
- `internal/skills/marketplace/clawhub/client.go` — ClawHub client (from task_08)
- `internal/skills/provenance.go` — WriteSidecar, ComputeHash (from task_06)
- `internal/skills/verify.go` — VerifyContent for post-install scanning

### Dependent Files
- `internal/config/config.go` — MarketplaceConfig read for client construction (from task_02)

### Related ADRs
- [ADR-003: Pluggable Registry Interface](adrs/adr-003.md) — CLI dispatches through interface, not ClawHub directly
- [ADR-004: Hash-Based Provenance](adrs/adr-004.md) — install writes sidecar with hash

## Deliverables
- Four new CLI subcommands in skill.go
- Updated newSkillCommand() registration
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for install + remove flow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] search command parses --limit flag correctly
  - [x] search command displays results in expected table format
  - [x] install command validates slug argument
  - [x] install with critical VerifyContent finding → error, skill not installed
  - [x] remove command refuses to delete non-marketplace skill (no sidecar)
  - [x] remove command deletes entire skill directory
  - [x] update --all flag triggers update for all marketplace skills
  - [x] update with no newer version → "already up to date" message
- Integration tests:
  - [x] install from mock HTTP server → skill directory created, SKILL.md present, .agh-meta.json written
  - [x] install + remove → directory cleaned up, skill no longer in registry
  - [x] install with clean content → sidecar hash matches SKILL.md SHA-256
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- All four commands registered and working end-to-end
- Install flow: download → extract → verify → hash → sidecar → success

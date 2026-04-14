---
status: pending
title: Migrate skill CLI to MultiRegistry and remove marketplace package
type: refactor
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 05: Migrate skill CLI to MultiRegistry and remove marketplace package

## Overview

Refactor the existing `agh skill search`, `install`, and `update` commands in `internal/cli/skill_commands.go` to use the new `MultiRegistry` and `Installer` instead of directly calling the old `marketplace.Registry` interface. The `install` command calls `Installer.Install()` for the domain-agnostic extraction, then performs the skill-specific step: writing the `.agh-meta.json` provenance sidecar via `skills.WriteSidecar()`. After all tests pass against the new implementation, delete the deprecated `internal/skills/marketplace/` package.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST refactor `agh skill search` to use `MultiRegistry.Search()` instead of direct ClawHub client
- MUST refactor `agh skill install` to use `Installer.Install()` for extraction, then `skills.WriteSidecar()` for provenance
- MUST refactor `agh skill update` to use `MultiRegistry.CheckUpdate()` with `versionIsNewer()`
- MUST NOT break `agh skill list`, `view`, `info`, `create` (unchanged commands)
- All existing skill marketplace integration tests MUST pass against new implementation
- MUST delete `internal/skills/marketplace/` package ONLY after gate passes
- Gate: all unit + integration tests from `skill_marketplace_integration_test.go` pass with new code path
- MUST pass `make verify` after completion (including after package deletion)
</requirements>

## Subtasks
- [ ] 5.1 Refactor `skill_commands.go` search/install/update to use MultiRegistry and Installer
- [ ] 5.2 Update skill install to call `Installer.Install()` then `skills.WriteSidecar()` for `.agh-meta.json`
- [ ] 5.3 Run full test suite against new implementation — verify all existing tests pass
- [ ] 5.4 Remove `internal/skills/marketplace/` package (registry.go, types.go, clawhub/)
- [ ] 5.5 Clean up any remaining imports of the deleted package
- [ ] 5.6 Run `make verify` to confirm clean build after deletion

## Implementation Details

See TechSpec "Build Order Steps 10-11" and "Updated CLI Commands — Skills" sections.

### Relevant Files
- `internal/cli/skill_commands.go:16-31` — `newSkillCommand` and subcommand registration
- `internal/cli/skill_marketplace.go` — Existing install/search/update logic (callers being replaced)
- `internal/cli/skill_marketplace_integration_test.go` — Gate tests that must pass before deletion
- `internal/skills/marketplace/registry.go:7-12` — `marketplace.Registry` interface being replaced
- `internal/skills/marketplace/clawhub/client.go` — Client being replaced by `internal/registry/clawhub/`
- `internal/skills/provenance.go:17` — `WriteSidecar()` for `.agh-meta.json` (remains, used by CLI caller)
- `internal/registry/multi.go` — MultiRegistry from task_02
- `internal/registry/installer.go` — Installer from task_02
- `internal/registry/clawhub/` — ClawHub adapter from task_03

### Dependent Files
- `internal/cli/skill_commands.go` — Import paths change from marketplace to registry
- `internal/cli/skill_marketplace.go` — Functions replaced or removed
- `internal/skills/marketplace/` — Entire package deleted

### Related ADRs
- [ADR-001: Multi-Source RegistrySource Interface](adrs/adr-001.md) — Migration from marketplace.Registry to RegistrySource
- [ADR-002: Separate CLI Namespaces](adrs/adr-002.md) — Skills CLI remains separate

## Deliverables
- Updated `skill_commands.go` using MultiRegistry and Installer
- Updated `skill_marketplace.go` (remaining non-extracted functions, or deleted if empty)
- Deleted `internal/skills/marketplace/` package
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests passing **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `agh skill search <query>` returns results via MultiRegistry (ClawHub source)
  - [ ] `agh skill search` with unreachable registry returns clear offline error
  - [ ] `agh skill install <slug>` calls Installer then writes `.agh-meta.json` sidecar
  - [ ] `agh skill install <slug>` with invalid archive returns extraction error
  - [ ] `agh skill update <name>` with newer version available triggers re-install
  - [ ] `agh skill update --check` shows update info without installing
  - [ ] `agh skill update <name>` with no update available shows "up to date" message
  - [ ] `agh skill list`, `view`, `info`, `create` remain unchanged and working
- Integration tests:
  - [ ] Full install flow: search → install → verify sidecar exists → list shows skill → remove
  - [ ] Install replaces existing skill (atomic move with backup)
  - [ ] All existing `skill_marketplace_integration_test.go` tests pass against new code
- Post-deletion verification:
  - [ ] `make verify` passes with `internal/skills/marketplace/` deleted
  - [ ] No remaining imports of `internal/skills/marketplace` in codebase
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- `internal/skills/marketplace/` package fully removed from codebase
- No import of deleted package remains anywhere
- Skill search/install/update behavior identical to before migration

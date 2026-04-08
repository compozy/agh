---
status: completed
title: "Extend Registry with marketplace source, provenance, and quarantine"
type: backend
complexity: high
dependencies:
  - task_03
  - task_06
---

# Task 7: Extend Registry with marketplace source, provenance, and quarantine

## Overview

Extend the skills registry to handle marketplace-sourced skills: detect skills with `.agh-meta.json` sidecars, tag them as `SourceMarketplace`, load provenance metadata, verify hashes on every load, and implement quarantine/block-on-load behavior for critically flagged marketplace skills. This is the integration point where types, loader, and provenance converge.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST distinguish marketplace skills (has .agh-meta.json sidecar) from user skills in `loadGlobalSkills()`
- MUST tag sidecar-backed skills as `SourceMarketplace`, others as `SourceUser`
- MUST load Provenance from sidecar and set on Skill struct during skill parsing
- MUST recompute SHA-256 hash on every load and compare with stored Provenance.Hash
- MUST log hash mismatch at Warn level with skill_name, expected_hash, actual_hash
- MUST re-run VerifyContent on hash mismatch
- MUST block marketplace skills with critical VerifyContent findings (block-on-load, consistent with existing processSkill behavior)
- MUST ensure BuildCatalog filters disabled skills (currently it does not — add filtering)
- MUST populate Skill.InstalledFrom from Provenance.Slug
</requirements>

## Subtasks
- [x] 7.1 Modify loadDirectorySkills to detect .agh-meta.json and set SourceMarketplace
- [x] 7.2 Load Provenance from sidecar during skill parsing, set on Skill struct
- [x] 7.3 Add hash verification in processSkill for marketplace skills
- [x] 7.4 Implement block-on-load for critically flagged marketplace skills
- [x] 7.5 Add disabled-skill filtering to BuildCatalog
- [x] 7.6 Write unit and integration tests for all marketplace registry flows

## Implementation Details

Changes span `internal/skills/registry.go` and `internal/skills/catalog.go`. The registry's `loadDirectorySkills()` and `processSkill()` paths need marketplace-aware logic.

See TechSpec "Integration Points > 2. Registry" section. Note the Codex-identified gap: `processSkill()` currently drops critically flagged skills entirely. For marketplace, this is the correct behavior (block-on-load).

### Relevant Files
- `internal/skills/registry.go` — loadGlobalSkills (line 211), loadDirectorySkills (line 276), processSkill (line 312)
- `internal/skills/catalog.go` — BuildCatalog (line 59) — needs disabled skill filtering
- `internal/skills/provenance.go` — ReadSidecar, VerifyHash, HasSidecar (from task_06)
- `internal/skills/loader.go` — ParseSkillFile now populates MCPServers/Hooks (from task_03)

### Dependent Files
- `internal/skills/mcp.go` — MCPResolver checks SourceMarketplace for consent (task_04)
- `internal/daemon/boot.go` — no changes needed, registry wiring unchanged

### Related ADRs
- [ADR-004: Hash-Based Provenance](adrs/adr-004.md) — defines hash verification and quarantine behavior

## Deliverables
- Extended registry.go with marketplace source handling and provenance verification
- Updated catalog.go to filter disabled skills
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for marketplace skill lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Skill with .agh-meta.json sidecar tagged as SourceMarketplace
  - [x] Skill without .agh-meta.json in same directory tagged as SourceUser
  - [x] Provenance loaded and set on Skill struct from sidecar
  - [x] Hash match → skill loads normally, no warnings
  - [x] Hash mismatch → VerifyContent re-run, warning logged
  - [x] Hash mismatch + critical VerifyContent → skill blocked (not loaded)
  - [x] Hash mismatch + clean VerifyContent → skill loaded with hash warning only
  - [x] InstalledFrom populated from Provenance.Slug
  - [x] BuildCatalog excludes disabled skills from XML output
  - [x] Existing tests still pass (no regression in bundled/user/workspace flows)
- Integration tests:
  - [x] Install marketplace skill (write sidecar), reload registry, verify skill appears as SourceMarketplace
  - [x] Tamper with SKILL.md after install, reload, verify hash warning logged
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Marketplace skills correctly identified and provenance-verified on every load
- BuildCatalog respects Enabled field

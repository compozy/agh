---
status: pending
title: XML catalog builder and CatalogProvider
type: backend
complexity: medium
dependencies:
  - task_03
---

# Task 04: XML catalog builder and CatalogProvider

## Overview

Implement the XML catalog builder that generates the `<available-skills>` block for system prompt injection, and the `CatalogProvider` that implements `session.PromptProvider` to integrate with the composed prompt assembly pipeline.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/catalog.go` with `BuildCatalog([]*Skill) string` and `CatalogProvider`
- MUST implement `CatalogProvider` satisfying `session.PromptProvider` interface
- MUST call `registry.ForWorkspace()` in `PromptSection()` to produce workspace-scoped catalogs
- MUST truncate skill descriptions at 200 characters
- MUST sort skills alphabetically by name in the catalog
- MUST return empty string when no skills are available
- MUST include usage instructions (`agh skill view <name>`) after the catalog
- MUST NOT cache catalog at provider level in Inc 1 — ForWorkspace handles caching
- MUST escape `<`, `>`, `&` in skill names and descriptions for XML-like output
</requirements>

## Subtasks
- [ ] 4.1 Implement `BuildCatalog()` that generates XML-like catalog string from skill list
- [ ] 4.2 Implement description truncation and XML character escaping
- [ ] 4.3 Implement `CatalogProvider` struct with `PromptSection()` method
- [ ] 4.4 Write unit tests for catalog generation and CatalogProvider

## Implementation Details

See TechSpec "Catalog XML Format" section for the exact output format. The CatalogProvider delegates to `registry.ForWorkspace()` for workspace-scoped skill resolution.

Note: The `session.PromptProvider` interface must exist before this task can fully compile (task_07), but the catalog logic itself only depends on the Registry (task_03). Implement against the expected interface signature.

### Relevant Files
- `internal/skills/registry.go` — ForWorkspace() provides the skill list (task_03)
- `internal/skills/types.go` — Skill and SkillMeta types (task_01)

### Dependent Files
- `daemon/daemon.go` — Will create CatalogProvider at boot (task_10)
- `cli/skill.go` — CLI view command uses similar formatting (task_11)

## Deliverables
- `internal/skills/catalog.go` with BuildCatalog and CatalogProvider
- `internal/skills/catalog_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] BuildCatalog produces valid XML-like format with skill names and descriptions
  - [ ] Descriptions truncated at 200 characters with ellipsis
  - [ ] Skills sorted alphabetically by name
  - [ ] Empty skill list produces empty string
  - [ ] Special characters in names (`<`, `>`, `&`) are escaped
  - [ ] Usage instructions appended after catalog
  - [ ] CatalogProvider returns empty string for workspace with no skills
  - [ ] CatalogProvider returns valid catalog for workspace with skills
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Catalog output matches TechSpec format exactly

---
status: completed
title: Skills registry delegation to Resolver
type: backend
complexity: medium
dependencies:
  - task_03
---

# Task 07: Skills registry delegation to Resolver

## Overview

Refactor `internal/skills` so workspace skill discovery is driven by `ResolvedWorkspace` skill paths from the Resolver instead of independent directory scans. Eliminates duplicate scanning and keeps a single source of truth for multi-root layouts (ADR-003).

<critical>
- READ `_techspec.md` skills impact row
- COORDINATE with task_03 outputs (`SkillPath` / resolved paths)
- TESTS REQUIRED — update `registry_test.go`
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST change `Registry.ForWorkspace` (or replacement API) to accept Resolver-provided path list or `ResolvedWorkspace` snapshot
- MUST preserve existing cache/TTL behavior where still applicable; avoid double full-directory scan
- MUST update all callers (`catalog`, memory assembler, daemon) to pass Resolver output
- MUST update tests: `TestRegistryForWorkspaceMergesGlobalAndWorkspaceSkills`, cache tests, and error paths
</requirements>

## Subtasks
- [x] 7.1 Design minimal API change between Resolver and Registry (avoid cyclic imports — use interfaces or narrow types)
- [x] 7.2 Implement path-based loading from Resolver output
- [x] 7.3 Update `catalog.go` and any `ForWorkspace` callers
- [x] 7.4 Adjust tests for new signatures; add regression for no double-scan if observable
- [x] 7.5 Verify skills package still meets coverage threshold

## Implementation Details

See TechSpec Impact Analysis `internal/skills/` and Development Sequencing step 8. If import cycles appear, define a narrow interface in `skills` or pass concrete `[]SkillPath` from `workspace`.

### Relevant Files
- `internal/skills/registry.go` — `ForWorkspace` and cache
- `internal/skills/catalog.go` — Catalog assembly
- `internal/skills/registry_test.go` — Extensive workspace tests

### Dependent Files
- `internal/daemon/` — Wires registry with Resolver (task_06)
- `internal/session/` — Prompt assembly may touch skills

### Related ADRs
- [ADR-003: Config from Root Only, Agents/Skills from All Dirs](adrs/adr-003.md)

## Deliverables
- Registry uses Resolver-supplied paths; duplicate scanning removed
- Updated unit tests in `internal/skills/` **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Skills merged from global + workspace + additional dirs match Resolver-provided paths
  - [x] Cache hit/miss behavior still correct when underlying files change (mtime)
  - [x] `ForWorkspace` with canceled context returns context error
- Integration tests:
  - [ ] Optional: covered by Resolver + registry integration if added
- Test coverage target: >=80% for `internal/skills`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/skills`
- `make verify` passes

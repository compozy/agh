---
status: pending
title: Bundled skills via go:embed
type: backend
complexity: low
dependencies:
  - task_01
---

# Task 05: Bundled skills via go:embed

## Overview

Create the bundled skills that ship with the AGH binary via `go:embed`. These 3 AGH-focused starter skills demonstrate the SKILL.md format and provide immediate value by teaching users how to use AGH's core features (sessions, memory, agent configuration).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/bundled/` directory with `embed.go` and skill files
- MUST use `//go:embed skills/**/SKILL.md` directive to embed skill files
- MUST export an `fs.FS` accessor for the embedded filesystem
- MUST create 3 bundled skills: `agh-session-guide`, `agh-memory-guide`, `agh-agent-setup`
- MUST write each skill as a valid SKILL.md with proper YAML frontmatter
- MUST ensure all bundled skills parse successfully via `ParseSkillFile()` in tests
- Each SKILL.md SHOULD contain actionable, AGH-specific instructions (not placeholder text)
</requirements>

## Subtasks
- [ ] 5.1 Create `internal/skills/bundled/embed.go` with go:embed directive and FS accessor
- [ ] 5.2 Write `agh-session-guide` SKILL.md (session lifecycle, CLI commands, session types)
- [ ] 5.3 Write `agh-memory-guide` SKILL.md (memory system, scopes, CLI commands, consolidation)
- [ ] 5.4 Write `agh-agent-setup` SKILL.md (AGENT.md format, providers, MCP servers, permissions)
- [ ] 5.5 Write unit tests verifying all bundled skills parse correctly

## Implementation Details

See TechSpec "Bundled Skills (F9)" section and ADR-005. The directory structure follows the pattern described in the TechSpec.

Skill content should reference actual AGH CLI commands and config patterns from the existing codebase.

### Relevant Files
- `internal/skills/loader.go` — ParseSkillFile used to validate bundled skills (task_01)
- `internal/config/agent.go` — AGENT.md format reference for agh-agent-setup skill content
- `internal/cli/session.go` — Session CLI commands reference for agh-session-guide content
- `internal/cli/memory.go` — Memory CLI commands reference for agh-memory-guide content

### Dependent Files
- `internal/skills/registry.go` — Will load bundled FS via RegistryConfig.BundledFS (task_03)

### Related ADRs
- [ADR-005: go:embed for Bundled Skills via fs.FS Interface](../adrs/adr-005.md) — go:embed with fs.FS for testability

## Deliverables
- `internal/skills/bundled/embed.go` with go:embed directive
- `internal/skills/bundled/skills/agh-session-guide/SKILL.md`
- `internal/skills/bundled/skills/agh-memory-guide/SKILL.md`
- `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`
- `internal/skills/bundled/bundled_test.go` with parse validation tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] All 3 bundled SKILL.md files parse successfully via ParseSkillFile
  - [ ] Each skill has required frontmatter fields (name, description)
  - [ ] Embedded FS contains expected directory structure
  - [ ] Content is non-empty for all skills
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- All 3 bundled skills contain actionable AGH-specific content
- Skills are accessible via the exported fs.FS accessor

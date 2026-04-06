---
status: pending
title: Core types and SKILL.md loader
type: backend
complexity: medium
dependencies: []
---

# Task 01: Core types and SKILL.md loader

## Overview

Define the foundational types for the skills system and implement the SKILL.md parser that extracts YAML frontmatter and Markdown body from skill files. This is the base layer that all other skills tasks depend on — every component in `internal/skills/` imports these types.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/` package with `types.go` and `loader.go`
- MUST define `Skill`, `SkillMeta`, `SkillSource`, `RegistryConfig`, `fileSnapshot` types per TechSpec "Core Interfaces" section
- MUST implement `ParseSkillFile(path string) (*Skill, error)` that reads a SKILL.md file and returns a parsed Skill
- MUST implement `parseFrontmatter(content string) (SkillMeta, string, error)` that splits YAML frontmatter from Markdown body
- MUST implement `scanDirectory(dir string) ([]string, error)` that finds all SKILL.md files under a root directory
- MUST use `gopkg.in/yaml.v3` for YAML parsing — install via `go get`
- MUST implement lenient parsing: warn on issues, skip only if name missing or YAML unparseable
- MUST respect scanning constraints: max depth 4, max 300 candidates per root, skip `.git/`, `node_modules/`, hidden dirs (except `.agh/`, `.agents/`)
- MUST use `fs.FS` interface (not concrete `embed.FS`) in `RegistryConfig.BundledFS` for testability
</requirements>

## Subtasks
- [ ] 1.1 Create `internal/skills/types.go` with all domain types
- [ ] 1.2 Add `gopkg.in/yaml.v3` dependency via `go get`
- [ ] 1.3 Implement YAML frontmatter parser with lenient error handling
- [ ] 1.4 Implement `ParseSkillFile()` that reads file, splits frontmatter, and returns Skill
- [ ] 1.5 Implement `scanDirectory()` with depth/count limits and directory skip rules
- [ ] 1.6 Write comprehensive unit tests for loader and types

## Implementation Details

Create the new `internal/skills/` package. See TechSpec "Core Interfaces" section for type definitions and "Loading Hierarchy" for scanning constraints.

The frontmatter parser should handle the `---` delimiter convention: content between first and second `---` lines is YAML, everything after is the Markdown body. Follow the AgentSkills spec for required/optional fields.

### Relevant Files
- `internal/config/agent.go` — Existing YAML frontmatter parser for AGENT.md files (similar pattern to follow)
- `internal/memory/store.go` — Existing package structure pattern to follow
- `go.mod` — Module path: `github.com/pedronauck/agh`

### Dependent Files
- `internal/skills/registry.go` — Will import types and loader (task_03)
- `internal/skills/verify.go` — Will import types (task_02)
- `internal/skills/catalog.go` — Will import types (task_04)

### Related ADRs
- [ADR-005: go:embed for Bundled Skills via fs.FS Interface](../adrs/adr-005.md) — Requires `fs.FS` in RegistryConfig

## Deliverables
- `internal/skills/types.go` with all domain types
- `internal/skills/loader.go` with ParseSkillFile, parseFrontmatter, scanDirectory
- `internal/skills/loader_test.go` with comprehensive unit tests
- `gopkg.in/yaml.v3` added to go.mod
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Parse valid SKILL.md with all frontmatter fields (name, description, version, metadata.agh)
  - [ ] Parse SKILL.md with only required fields (name, description)
  - [ ] Handle malformed YAML (return error, do not panic)
  - [ ] Handle missing frontmatter delimiters (return error)
  - [ ] Handle empty Markdown body (valid — body can be empty)
  - [ ] Handle missing `name` field (skip with error)
  - [ ] Handle very long content (>50K chars, should still parse)
  - [ ] Lenient parsing: warn on unknown fields, still parse successfully
  - [ ] scanDirectory finds SKILL.md files at depth 1-4
  - [ ] scanDirectory stops at max depth 4
  - [ ] scanDirectory caps at 300 candidates per root
  - [ ] scanDirectory skips `.git/`, `node_modules/`, hidden dirs
  - [ ] scanDirectory does NOT skip `.agh/` and `.agents/` directories
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Types are used consistently across the package
- Lenient parser handles edge cases from other AgentSkills tools gracefully

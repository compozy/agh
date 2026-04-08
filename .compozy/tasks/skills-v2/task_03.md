---
status: completed
title: "Parse metadata.agh fields in loader"
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 3: Parse metadata.agh fields in loader

## Overview

Extend the skill loader to extract `metadata.agh.mcp_servers` and `metadata.agh.hooks` from the free-form `Metadata map[string]any` into typed `[]MCPServerDecl` and `[]HookDecl` fields on the `Skill` struct. This enables skills to declare MCP server dependencies and lifecycle hooks in their YAML frontmatter.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `parseAGHMetadata(skill *Skill)` function to extract typed fields from `skill.Meta.Metadata["agh"]`
- MUST call `parseAGHMetadata` after `parseSkillContent()` in both `ParseSkillFile()` and `parseBundledSkill()`
- MUST use type assertions on `map[string]any` to extract nested structures — no additional YAML parsing
- MUST gracefully ignore missing or malformed `metadata.agh` fields (log warning, don't fail)
- MUST validate HookEvent values against known constants (reject unknown events with warning)
- MUST validate MCPServerDecl has non-empty Name and Command
- SHOULD handle environment variable `${}` syntax in MCPServerDecl.Env values (store raw, resolve later)
</requirements>

## Subtasks
- [x] 3.1 Implement `parseAGHMetadata(skill *Skill)` function in loader.go
- [x] 3.2 Wire parseAGHMetadata into ParseSkillFile and parseBundledSkill code paths
- [x] 3.3 Add validation for MCPServerDecl (name, command required) and HookDecl (valid event)
- [x] 3.4 Write unit tests with fixture SKILL.md files containing metadata.agh fields

## Implementation Details

Changes are in `internal/skills/loader.go`. The existing `parseSkillContent()` already unmarshals `metadata` into `map[string]any` via `decodeSkillMeta()`. The new function type-asserts nested maps.

See TechSpec "Integration Points > 1. Loader" section.

### Relevant Files
- `internal/skills/loader.go` — ParseSkillFile (line 36), parseSkillContent (line 175), decodeSkillMeta (line 155)
- `internal/skills/types.go` — MCPServerDecl, HookDecl types (from task_01)

### Dependent Files
- `internal/skills/registry.go` — processSkill will see populated MCPServers/Hooks on Skill
- `internal/skills/mcp.go` — MCPResolver reads skill.MCPServers (task_04)
- `internal/skills/hooks.go` — HookRunner reads skill.Hooks (task_05)

## Deliverables
- `parseAGHMetadata()` function in loader.go
- Updated ParseSkillFile and parseBundledSkill to call it
- Fixture SKILL.md files in testdata/ with metadata.agh sections
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] SKILL.md with `metadata.agh.mcp_servers` populates skill.MCPServers correctly
  - [x] SKILL.md with `metadata.agh.hooks` populates skill.Hooks correctly
  - [x] SKILL.md with both mcp_servers and hooks populates both
  - [x] SKILL.md without metadata.agh leaves MCPServers and Hooks nil
  - [x] Malformed metadata.agh (wrong type) logs warning and leaves fields nil
  - [x] MCPServerDecl with missing name is rejected with warning
  - [x] MCPServerDecl with missing command is rejected with warning
  - [x] HookDecl with unknown event string is rejected with warning
  - [x] HookDecl with valid event and timeout parses correctly
  - [x] Environment variables with `${}` syntax stored raw in Env map
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Existing loader tests still pass (no regression)

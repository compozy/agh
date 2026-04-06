---
status: pending
title: CLI skill commands
type: backend
complexity: medium
dependencies:
  - task_01
  - task_02
  - task_03
  - task_04
  - task_08
---

# Task 11: CLI skill commands

## Overview

Implement the `agh skill` CLI subcommand group with `list`, `view`, `info`, and `create` commands. All commands operate locally on the filesystem — no daemon connection required. They instantiate an ephemeral registry with the same resolution and verification pipeline to ensure consistency with the daemon catalog.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `cli/skill.go` with `newSkillCommand(deps)` returning cobra.Command
- MUST register `agh skill` in `root.go` via `cmd.AddCommand(newSkillCommand(deps))`
- MUST implement `agh skill list [--source <source>]` showing name, description, source, enabled status
- MUST implement `agh skill view <name> [--file <path>]` returning skill content in XML-like format
- MUST implement `agh skill info <name>` showing full metadata, source, path, resource listing
- MUST implement `agh skill create [name]` scaffolding a new skill directory with template SKILL.md
- ALL commands MUST be local-only — no daemon client connection
- MUST reuse `skills.Registry` with same `LoadAll()` + `ForWorkspace()` + `VerifyContent()` pipeline
- MUST exclude security-blocked skills from `view` output
- MUST support human, JSON, and toon output formats (existing `writeCommandOutput` pattern)
- `agh skill view` MUST output XML-like delimiters for LLM consumption (not strict XML)
</requirements>

## Subtasks
- [ ] 11.1 Create `cli/skill.go` with `newSkillCommand()` parent and 4 subcommands
- [ ] 11.2 Implement `list` command with source filtering and output formatting
- [ ] 11.3 Implement `view` command with XML-like output and optional --file flag
- [ ] 11.4 Implement `info` command with full metadata display
- [ ] 11.5 Implement `create` command that scaffolds skill directory + template SKILL.md
- [ ] 11.6 Register skill command in `root.go`
- [ ] 11.7 Write unit tests for all commands

## Implementation Details

Follow existing CLI patterns from `cli/memory.go` and `cli/agent.go`. Use `commandDeps.loadConfig()` and `commandDeps.resolveHome()` for directory resolution. See TechSpec "CLI (internal/cli)" section.

The `view` command output format follows TechSpec "`agh skill view` Output Format" section — XML-like delimiters with raw Markdown body, not strict XML.

### Relevant Files
- `internal/cli/root.go` — Command registration pattern (line 78-85)
- `internal/cli/memory.go` — Local CLI command pattern reference
- `internal/cli/agent.go` — Agent list/info command pattern reference
- `internal/cli/format.go` — Output format helpers

### Dependent Files
- `internal/cli/root.go` — Add `cmd.AddCommand(newSkillCommand(deps))`

## Deliverables
- `internal/cli/skill.go` with all 4 subcommands
- Modified `internal/cli/root.go` with skill command registration
- `internal/cli/skill_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `agh skill list` returns all visible skills with correct fields
  - [ ] `agh skill list --source bundled` filters by source
  - [ ] `agh skill view <name>` returns XML-like formatted content
  - [ ] `agh skill view <name>` excludes security-blocked skills
  - [ ] `agh skill view <name> --file <path>` returns specific resource file
  - [ ] `agh skill view` with unknown skill returns error
  - [ ] `agh skill info <name>` shows metadata, source, path, resources
  - [ ] `agh skill create myskill` creates directory with template SKILL.md
  - [ ] `agh skill create` with existing name returns error
  - [ ] All commands work without daemon running
  - [ ] JSON output format produces valid JSON
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- All commands work without daemon running
- Security-blocked skills excluded from view output
- Output formats match existing CLI conventions

---
status: completed
domain: CLI/Integration
type: Feature Implementation
scope: Full
complexity: high
dependencies:
    - task_02
---

# Task 4: CLI Commands and ClawHub Client

## Overview

Implement all `agh skill` CLI subcommands and the ClawHub marketplace HTTP client. This includes local skill management (list, view, info, create, remove) and marketplace operations (search, install). The CLI commands use the skills Registry for local operations and the ClawHub client for remote marketplace interaction.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `agh skill` parent command with 7 subcommands: list, view, info, create, install, remove, search
- MUST follow existing CLI patterns (cobra commands, daemonCommandDeps injection, same structure as session/workgroup commands)
- MUST implement `view` command that returns skill body wrapped in `<skill_content>` XML with `<skill_resources>` listing
- MUST implement `view --file` flag to read specific files within a skill directory (with path traversal prevention)
- MUST implement ClawHub HTTP client with search and install endpoints
- MUST implement exponential backoff retry for ClawHub requests (1.5s → 30s max, 5 retries)
- MUST run VerifyContent() on skills downloaded from ClawHub before installing
- MUST implement `create` command that scaffolds a new skill directory with SKILL.md template
- MUST register the `skill` command group in the CLI root command
</requirements>

## Subtasks
- [x] 4.1 Create `internal/skills/clawhub.go` with ClawHub HTTP client (NewClient, Search, Install with retry logic)
- [x] 4.2 Create `internal/cli/skill.go` with parent command and all 7 subcommands
- [x] 4.3 Implement `list` — tabular output with name, description, source, enabled
- [x] 4.4 Implement `view` — return `<skill_content>` XML with body and resource listing; `--file` flag for specific files with path traversal guard
- [x] 4.5 Implement `info` — full metadata output (frontmatter, source, path, resources)
- [x] 4.6 Implement `create` — scaffold skill directory with SKILL.md template
- [x] 4.7 Implement `install` — download from ClawHub, verify, extract to ~/.agh/skills/
- [x] 4.8 Implement `remove` — delete skill directory from filesystem
- [x] 4.9 Implement `search` — query ClawHub and display results
- [x] 4.10 Register `skill` command in root.go

## Implementation Details

### Relevant Files
- `internal/skills/clawhub.go` — New file: marketplace client
- `internal/cli/skill.go` — New file: all CLI commands
- `internal/cli/root.go` — Modify: register skill command group (line ~30-40, following existing pattern)
- `internal/cli/session.go` — Reference: cobra subcommand group pattern (lines 11-24)
- `internal/cli/daemon.go` — Reference: daemonCommandDeps usage pattern

### Dependent Files
- `internal/skills/verify.go` (task_01) — used by install command for security scanning
- `internal/skills/registry.go` (task_02) — used by list/view/info/remove commands

### Related ADRs
- [ADR-003: System Prompt + CLI Access](adrs/adr-003.md) — Defines the `agh skill view` output format and `<skill_content>` XML structure

## Deliverables
- `internal/skills/clawhub.go` with ClawHub client
- `internal/cli/skill.go` with all 7 subcommands
- Modified `internal/cli/root.go` with skill command registration
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for CLI commands **(REQUIRED)**

## Tests
- Unit tests:
  - [x] ClawHub Search parses valid API response
  - [x] ClawHub Search handles empty results
  - [x] ClawHub Search retries on transient errors with backoff
  - [x] ClawHub Install downloads, verifies, and extracts skill
  - [x] ClawHub Install rejects skill with critical security warning
  - [x] `view` command returns body without frontmatter wrapped in `<skill_content>` XML
  - [x] `view` command includes `<skill_resources>` listing of files in skill directory
  - [x] `view --file` returns content of specific file within skill
  - [x] `view --file` rejects path traversal attempts (`../` and absolute paths)
  - [x] `list` command returns formatted table of all skills
  - [x] `list --source` filters by source
  - [x] `info` command returns full metadata
  - [x] `create` scaffolds directory with valid SKILL.md template
  - [x] `remove` deletes skill directory
  - [x] `search` displays ClawHub results in formatted output
- Integration tests:
  - [x] End-to-end: create skill → list shows it → view returns content → remove deletes it
  - [x] ClawHub integration with HTTP test server (search → install → verify)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All 7 CLI commands work correctly
- ClawHub client handles search, install, and error scenarios
- Path traversal is prevented in `view --file`
- Downloaded skills are verified before installation
- `make verify` passes (fmt + lint + test + build)

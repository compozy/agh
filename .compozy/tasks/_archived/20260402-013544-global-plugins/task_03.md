---
status: completed
domain: Core/Drivers
type: Refactor
scope: Full
complexity: high
dependencies:
    - task_01
    - task_02
---

# Task 3: All Drivers — Zero Workdir Writes

## Overview

Modify all four driver `Start()` and `buildCommand()` methods to eliminate every file write to the user's working directory. Hooks come from global plugins (task_01). System prompts and runtime config are passed via CLI flags and environment variables instead of files. After this task, AGH writes zero files to the user's project directory.

<critical>
- ALWAYS READ the TechSpec before starting
- CHECK each driver's CLI documentation for exact flag syntax before implementing
- REFERENCE ADR-005 for the specific replacement mechanism per driver
- TESTS REQUIRED — every task MUST include tests in deliverables
- Verify each driver's buildCommand() output matches what the target CLI expects
</critical>

<requirements>
- MUST remove all os.MkdirAll, os.WriteFile, and file-write logic from all 4 driver Start() methods
- MUST remove all BuildHookConfig() calls from Start() methods (method already removed in task_02)
- Claude: MUST add `--bare` flag to buildCommand() and remove `--settings` flag
- Codex: MUST replace `.codex/AGENTS.md` write with `-c 'developer_instructions=...'` flag
- Codex: MUST remove `.codex/hooks.json` write and `--enable codex_hooks` flag
- OpenCode: MUST replace `opencode.json` file write with `OPENCODE_CONFIG_CONTENT` env var in buildEnv()
- Pi: MUST replace `.pi/SYSTEM.md` write with `--system-prompt` flag
- Pi: MUST replace `.pi/AGENTS.md` write with `--append-system-prompt` flag
- Pi: MUST remove `.pi/extensions/agh-hook.ts` write
- MUST preserve all non-file-write logic in Start() (PTY setup, process management, etc.)
- MUST keep ParseHookEvent() unchanged in all drivers
- MUST keep buildAdditionalContext() in Pi driver (used for --append-system-prompt content)
- MUST keep buildConfig()/buildConfigJSON() in OpenCode driver (used for OPENCODE_CONFIG_CONTENT)
</requirements>

## Subtasks
- [x] 3.1 Claude driver — remove file writes from Start(), add `--bare` to buildCommand(), remove `--settings`
- [x] 3.2 Codex driver — remove AGENTS.md and hooks.json writes from Start(), add `-c developer_instructions` to buildCommand(), remove `--enable codex_hooks`
- [x] 3.3 OpenCode driver — remove opencode.json write from Start(), add OPENCODE_CONFIG_CONTENT to buildEnv()
- [x] 3.4 Pi driver — remove SYSTEM.md, AGENTS.md, agh-hook.ts writes from Start(), add `--system-prompt` and `--append-system-prompt` to buildCommand()
- [x] 3.5 Update all 4 driver test files to verify no file writes occur and new flags/env vars are correct
- [x] 3.6 Remove helper functions that are now dead code (e.g., resolveConfigPath if unused)

## Implementation Details

### Relevant Files
- `internal/drivers/claude/claude.go` — Modify Start() (lines 177-244) and buildCommand() (lines 515-545)
- `internal/drivers/codex/codex.go` — Modify Start() (lines 168-239) and buildCommand() (lines 454-472)
- `internal/drivers/opencode/opencode.go` — Modify Start() (lines 249-339) and buildEnv() (lines 1119-1144)
- `internal/drivers/pi/pi.go` — Modify Start() (lines 175-252) and buildCommand() (lines 471-494)

### Dependent Files
- `internal/drivers/claude/claude_test.go` — Update Start/buildCommand tests
- `internal/drivers/codex/codex_test.go` — Update Start/buildCommand tests
- `internal/drivers/opencode/opencode_test.go` — Update Start/buildEnv tests
- `internal/drivers/pi/pi_test.go` — Update Start/buildCommand tests

### Related ADRs
- [ADR-005: Zero Workdir Pollution via CLI Flags and Environment Variables](adrs/adr-005.md) — Defines exact replacement per driver
- [ADR-001: Global Plugins Over Per-Session File Generation](adrs/adr-001.md) — Hooks come from global plugins now

## Deliverables
- All 4 drivers write zero files to user's workdir
- Claude uses `--bare --system-prompt --allowedTools`
- Codex uses `-c 'developer_instructions="..."'`
- OpenCode uses `OPENCODE_CONFIG_CONTENT` env var
- Pi uses `--system-prompt` and `--append-system-prompt`
- Updated tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests per driver:
  - [x] Claude: `buildCommand()` includes `--bare` and does NOT include `--settings`
  - [x] Claude: `buildCommand()` still includes `--system-prompt`, `--allowedTools`, `--model`, `--name`
  - [x] Claude: `Start()` does not call os.MkdirAll or os.WriteFile
  - [x] Codex: `buildCommand()` includes `-c` with developer_instructions and does NOT include `--enable codex_hooks`
  - [x] Codex: `Start()` does not write `.codex/AGENTS.md` or `.codex/hooks.json`
  - [x] OpenCode: `buildEnv()` includes `OPENCODE_CONFIG_CONTENT` with valid JSON containing model, agent, tools
  - [x] OpenCode: `Start()` does not write `opencode.json`
  - [x] Pi: `buildCommand()` includes `--system-prompt` with system prompt content
  - [x] Pi: `buildCommand()` includes `--append-system-prompt` with runtime context
  - [x] Pi: `Start()` does not write `.pi/SYSTEM.md`, `.pi/AGENTS.md`, or `.pi/extensions/agh-hook.ts`
  - [x] Pi: `buildAdditionalContext()` still produces correct runtime context string
  - [x] OpenCode: `buildConfig()` still produces correct JSON config structure
- Integration tests:
  - [x] Verify no files exist in t.TempDir() workdir after driver Start() for each driver
- Test coverage target: >=80%

## Success Criteria
- Zero calls to os.WriteFile or os.MkdirAll in any driver Start() method
- Each driver's buildCommand()/buildEnv() produces the correct flags/env vars per TechSpec
- All ParseHookEvent() methods still work unchanged
- All existing integration tests pass
- `make verify` passes (fmt + lint + test + build)

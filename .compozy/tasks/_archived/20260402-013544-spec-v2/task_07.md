---
status: completed
title: Claude Code Driver
type: ""
complexity: medium
dependencies:
    - task_01
    - task_06
---

# Task 7: Claude Code Driver

## Overview
Implement the first AgentDriver for Claude Code, validating the driver interface design. This includes command building with correct CLI flags, settings.json hook configuration generation, hook event parsing into normalized HookEvent, readiness detection via PTY output pattern matching, and tool name translation.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement the full AgentDriver interface per docs/spec-v2/05-drivers.md
- MUST build the claude command with flags: --dangerously-skip-permissions, --model, --name, --settings, --add-dir, --system-prompt, --allowedTools
- MUST generate settings.json with PostToolUse hooks per docs/spec-v2/05-drivers.md
- MUST parse Claude hook event JSON into normalized HookEvent per docs/spec-v2/08-data-models.md
- MUST implement readiness detection by scanning PTY output buffer for Claude prompt pattern
- MUST translate kernel canonical tool names to Claude PascalCase names (read→Read, write→Write, etc.)
- MUST implement health check via PID existence (syscall.Kill(pid, 0))
- MUST implement graceful stop: close PTY fd → SIGTERM → timeout → SIGKILL
- MUST use PtyAllocator interface for testability (inject MockPtyAllocator in tests)
</requirements>

## Subtasks
- [x] 7.1 Implement ClaudeDriver struct with all AgentDriver methods
- [x] 7.2 Implement command building with correct flags and environment variables
- [x] 7.3 Implement settings.json hook config generation (BuildHookConfig)
- [x] 7.4 Implement hook event JSON parsing and normalization (ParseHookEvent)
- [x] 7.5 Implement readiness detection via PTY output buffer scanning (DetectReady)
- [x] 7.6 Implement tool name translation (kernel canonical → Claude PascalCase)

## Implementation Details
Refer to docs/spec-v2/05-drivers.md for the complete Claude Code driver spec including start sequence, hook format, and tool mapping table.

### Relevant Files
- `docs/spec-v2/05-drivers.md` — Claude driver spec, start sequence, hook format
- `docs/spec-v2/08-data-models.md` — HookEvent, AgentHealth types
- `docs/spec-v2/11-testing.md` — MockPty for driver tests

### Dependent Files
- `internal/kernel/types.go` — AgentDriver interface, StartOpts, AgentProcess, HookEvent, HookConfig
- `internal/pty/` — PtyAllocator, MockPty from task_06

## Deliverables
- internal/drivers/claude/claude.go — full ClaudeDriver implementation
- internal/drivers/claude/claude_test.go — comprehensive test suite
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Command build includes correct flags: --dangerously-skip-permissions, --model, --name, --system-prompt, --allowedTools
  - [x] Model translation: "sonnet" → "sonnet", "opus" → "opus" (passthrough)
  - [x] Tool translation: kernel ["read","write","bash"] → Claude ["Read","Write","Bash"]
  - [x] Hook config generation: valid settings.json with PostToolUse matcher regex
  - [x] Hook event parsing: raw Claude JSON → normalized HookEvent with correct fields
  - [x] Readiness pattern: regex matches Claude prompt indicator in output buffer
  - [x] Readiness timeout: context deadline exceeded when prompt never appears
  - [x] Health check: returns Alive=true for existing PID, Alive=false for dead PID
  - [x] SendMessage: writes message + newline to MockPty
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- var _ AgentDriver = (*ClaudeDriver)(nil) compiles
- All test cases from docs/spec-v2/11-testing.md Claude Driver Tests section pass

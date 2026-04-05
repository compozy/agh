---
status: completed
title: Pi Driver
type: ""
complexity: medium
dependencies:
    - task_01
    - task_06
---

# Task 26: Pi Driver

## Overview
Implement the AgentDriver for Pi (badlogic), including .pi/SYSTEM.md system prompt generation, TypeScript extension file for hook events, --tools CSV flag translation from kernel canonical names, and TUI readiness detection.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement the full AgentDriver interface per docs/spec-v2/05-drivers.md
- MUST generate .pi/SYSTEM.md with system prompt content
- MUST optionally generate .pi/AGENTS.md with additional context
- MUST build command with flags: --cwd, --model, --tools (CSV list)
- MUST generate .pi/extensions/agh-hook.ts TypeScript extension for hook events
- MUST translate kernel tool names to Pi names (glob → find, list → ls)
- MUST detect readiness via Pi TUI prompt pattern in PTY output
- MUST implement health check via PID existence
- MUST use PtyAllocator for testability
</requirements>

## Subtasks
- [x] 26.1 Implement PiDriver struct with all AgentDriver methods
- [x] 26.2 Implement .pi/SYSTEM.md generation with system prompt
- [x] 26.3 Implement command building with --cwd, --model, --tools
- [x] 26.4 Implement TypeScript extension generation for hook events
- [x] 26.5 Implement tool name translation (kernel canonical → Pi names)
- [x] 26.6 Implement readiness detection via Pi TUI prompt pattern

## Implementation Details
Refer to docs/spec-v2/05-drivers.md Pi Driver section for complete spec.

### Relevant Files
- `docs/spec-v2/05-drivers.md` — Pi driver spec
- `docs/spec-v2/11-testing.md` — Pi driver test cases

### Dependent Files
- `internal/kernel/types.go` — AgentDriver interface
- `internal/pty/mock.go` — MockPty for tests

## Deliverables
- internal/drivers/pi/pi.go — full PiDriver implementation
- internal/drivers/pi/pi_test.go — comprehensive test suite
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Command build includes: --cwd, --model, --tools with correct CSV
  - [x] .pi/SYSTEM.md written with system prompt content
  - [x] TypeScript extension file generated with correct hook handler
  - [x] Hook event parsing: Pi extension event → normalized HookEvent
  - [x] Tool translation: kernel "glob" → Pi "find", kernel "list" → Pi "ls"
  - [x] Readiness: Pi TUI prompt pattern detected in output buffer
  - [x] SendMessage writes to MockPty correctly
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- var _ AgentDriver = (*PiDriver)(nil) compiles

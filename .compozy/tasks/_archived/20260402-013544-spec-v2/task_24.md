---
status: completed
domain: Drivers
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
    - task_06
---

# Task 24: Codex Driver

## Overview
Implement the AgentDriver for OpenAI Codex, including command building with Codex-specific flags, AGENTS.md system prompt file generation, hooks.json configuration, sandbox mode mapping for tool restrictions, and readiness detection via thread.started event.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement the full AgentDriver interface per docs/spec-v2/05-drivers.md
- MUST build command with flags: --yolo, -m, -C, --enable codex_hooks
- MUST generate AGENTS.md with system prompt content at {workdir}/.codex/AGENTS.md
- MUST generate hooks.json with PostToolUse matcher at {workdir}/.codex/hooks.json
- MUST map researcher type to --sandbox read-only
- MUST detect readiness via thread.started event in PTY output
- MUST implement health check via PID existence
- MUST use PtyAllocator for testability
</requirements>

## Subtasks
- [x] 24.1 Implement CodexDriver struct with all AgentDriver methods
- [x] 24.2 Implement command building with Codex-specific flags
- [x] 24.3 Implement AGENTS.md generation with system prompt
- [x] 24.4 Implement hooks.json generation
- [x] 24.5 Implement sandbox mode mapping (researcher → read-only)
- [x] 24.6 Implement readiness detection (thread.started pattern)

## Implementation Details
Refer to docs/spec-v2/05-drivers.md Codex Driver section for complete spec.

### Relevant Files
- `docs/spec-v2/05-drivers.md` — Codex driver spec
- `docs/spec-v2/11-testing.md` — Codex driver test cases

### Dependent Files
- `internal/kernel/types.go` — AgentDriver interface
- `internal/pty/mock.go` — MockPty for tests

## Deliverables
- internal/drivers/codex/codex.go — full CodexDriver implementation
- internal/drivers/codex/codex_test.go — comprehensive test suite
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Command build includes: --yolo, -m, -C, --enable codex_hooks
  - [x] AGENTS.md written with system prompt content
  - [x] hooks.json written with PostToolUse matcher
  - [x] Hook event parsing: raw Codex JSON → normalized HookEvent
  - [x] Sandbox mapping: researcher type → --sandbox read-only
  - [x] Readiness: thread.started pattern detected in output buffer
  - [x] SendMessage writes to MockPty correctly
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- var _ AgentDriver = (*CodexDriver)(nil) compiles

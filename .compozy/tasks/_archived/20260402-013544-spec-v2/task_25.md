---
status: completed
title: OpenCode Driver
type: ""
complexity: medium
dependencies:
    - task_01
    - task_06
---

# Task 25: OpenCode Driver

## Overview
Implement the AgentDriver for OpenCode (SST), including opencode.json configuration generation, dual-mode support (TUI and server), SSE event stream subscription for hook events, and health endpoint polling for server mode readiness.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement the full AgentDriver interface per docs/spec-v2/05-drivers.md
- MUST generate opencode.json with model, agent config, tools, and permissions
- MUST support two modes: TUI (opencode tui) and server (opencode serve)
- MUST implement SSE event stream goroutine for hook events (no file-based hooks)
- MUST detect readiness via health endpoint (server mode) or PTY output (TUI mode)
- MUST map researcher type to read-only tools in opencode.json
- MUST store mode in AgentProcess.Metadata
- MUST implement message delivery: PTY stdin (TUI) or HTTP POST (server)
- MUST use PtyAllocator for testability
</requirements>

## Subtasks
- [x] 25.1 Implement OpenCodeDriver struct with all AgentDriver methods
- [x] 25.2 Implement opencode.json generation with model, agent, tools, permissions
- [x] 25.3 Implement dual-mode command building (TUI vs server)
- [x] 25.4 Implement SSE event stream goroutine for hook ingestion
- [x] 25.5 Implement readiness detection (health endpoint or PTY output)
- [x] 25.6 Implement dual-mode message delivery (PTY stdin or HTTP POST)

## Implementation Details
Refer to docs/spec-v2/05-drivers.md OpenCode Driver section for complete spec.

### Relevant Files
- `docs/spec-v2/05-drivers.md` — OpenCode driver spec
- `docs/spec-v2/11-testing.md` — OpenCode driver test cases

### Dependent Files
- `internal/kernel/types.go` — AgentDriver interface
- `internal/pty/mock.go` — MockPty for tests

## Deliverables
- internal/drivers/opencode/opencode.go — full OpenCodeDriver implementation
- internal/drivers/opencode/opencode_test.go — comprehensive test suite
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] opencode.json generated with correct model, agent config, permissions, tools
  - [x] Server mode: correct health endpoint URL construction
  - [x] TUI mode: correct command building
  - [x] SSE event parsing: message.part.updated → normalized HookEvent
  - [x] Researcher type: read-only tools in config
  - [x] Mode stored in Metadata
  - [x] SendMessage: PTY stdin (TUI) or HTTP POST (server)
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- var _ AgentDriver = (*OpenCodeDriver)(nil) compiles

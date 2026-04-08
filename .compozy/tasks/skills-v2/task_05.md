---
status: completed
title: "Implement HookRunner subprocess dispatch"
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 5: Implement HookRunner subprocess dispatch

## Overview

Create the `HookRunner` that executes skill-declared lifecycle hooks as subprocess commands with JSON stdin/stdout protocol, configurable timeout, and fail-open semantics. Hooks are dispatched in skill source precedence order (bundled first, workspace last), alphabetical within same level.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `HookRunner` struct in `internal/skills/hooks.go` with logger
- MUST implement `RunHooks(ctx, event HookEvent, skills []*Skill, payload HookPayload) []HookResult`
- MUST define `HookPayload` (SessionID, AgentName, Workspace, Event) and `HookResult` (SkillName, Event, Output, Error, Duration)
- MUST execute hook commands via `exec.CommandContext` with JSON payload written to stdin
- MUST capture stdout as hook output string
- MUST enforce configurable timeout per hook (default 5s) via `context.WithTimeout`
- MUST implement fail-open: hook errors logged at Warn level, never returned as blocking errors
- MUST order hooks by source precedence (bundled → marketplace → user → additional → workspace), alphabetical within level
- MUST filter skills to only those with hooks matching the given event
</requirements>

## Subtasks
- [x] 5.1 Create HookRunner struct, HookPayload, HookResult types in hooks.go
- [x] 5.2 Implement hook filtering and precedence ordering
- [x] 5.3 Implement subprocess execution with JSON stdin, stdout capture, and timeout
- [x] 5.4 Implement fail-open error handling with structured logging
- [x] 5.5 Write unit tests with mock hook scripts

## Implementation Details

New file `internal/skills/hooks.go`. Uses `os/exec.CommandContext` for subprocess management. Hook scripts receive JSON via stdin and return output via stdout.

See TechSpec "HookRunner" and "Data Flow — Lifecycle Hooks" sections.

### Relevant Files
- `internal/skills/types.go` — HookDecl, HookEvent, SkillSource, Skill struct

### Dependent Files
- `internal/daemon/notifier.go` — hook dispatch phase calls HookRunner (task_09)

### Related ADRs
- [ADR-002: Hybrid Hook Execution Model](adrs/adr-002.md) — defines subprocess execution model and fail-open semantics

## Deliverables
- `internal/skills/hooks.go` with HookRunner, HookPayload, HookResult
- Test hook scripts in `internal/skills/testdata/`
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] RunHooks with skill having matching on_session_created hook → subprocess executed
  - [x] RunHooks with skill having on_session_stopped hook, called with on_session_created → hook not executed
  - [x] Hook receives correct JSON payload via stdin (SessionID, AgentName, Workspace, Event)
  - [x] Hook stdout captured in HookResult.Output
  - [x] Hook exceeding timeout → killed, HookResult.Error set, session not blocked
  - [x] Hook returning non-zero exit → HookResult.Error set, logged at Warn, session not blocked
  - [x] Multiple skills with hooks → executed in source precedence order
  - [x] Same-source skills → alphabetical order by skill name
  - [x] Skill with no matching hooks → skipped
  - [x] Empty skills list → empty results
  - [x] Hook with custom Env vars → environment variables passed to subprocess
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Fail-open behavior verified (hook failure never blocks caller)

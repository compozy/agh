---
status: completed
title: Integrate session, input, prompt, event, and agent dispatch
type: backend
complexity: high
dependencies:
  - task_06
  - task_09
---

# Task 10: Integrate session, input, prompt, event, and agent dispatch

## Overview

Wire typed dispatch calls into the session manager at all `session.*`, `input.*`, `prompt.*`, `event.*`, and `agent.*` lifecycle points. This transforms the session manager from notifying after-the-fact to dispatching sync barriers before operations (pre_create, pre_resume, pre_stop) and async observations after them.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `session.pre_create` sync dispatch before session creation — can block or patch
- MUST add `session.post_create` async dispatch after session activation
- MUST add `session.pre_resume` sync dispatch before session resume
- MUST add `session.post_resume` async dispatch after resume
- MUST add `session.pre_stop` sync dispatch before session stop
- MUST add `session.post_stop` async dispatch after session stopped
- MUST add `input.pre_submit` sync dispatch before user input is processed — can patch message
- MUST add `prompt.post_assemble` sync dispatch after prompt assembly — can patch assembled prompt
- MUST add `event.pre_record` and `event.post_record` async dispatch around event recording
- MUST add `agent.pre_start`, `agent.spawned`, `agent.crashed`, `agent.stopped` dispatch at agent lifecycle points
- MUST handle pre_create dispatch failure by aborting session creation with error
- MUST preserve existing session manager error handling and cleanup patterns
</requirements>

## Subtasks
- [x] 10.1 Add session lifecycle dispatch (pre/post create, resume, stop) in manager_lifecycle.go
- [x] 10.2 Add input.pre_submit dispatch in the input/prompt processing path
- [x] 10.3 Add prompt.post_assemble dispatch in the prompt assembly path
- [x] 10.4 Add event dispatch (pre/post record) in the event recording path
- [x] 10.5 Add agent lifecycle dispatch (pre_start, spawned, crashed, stopped)
- [x] 10.6 Write integration tests for session lifecycle hooks and permission escalation e2e

## Implementation Details

Modify existing files:
- `internal/session/manager_lifecycle.go` — Add dispatch calls around Create, Resume, Stop
- `internal/session/manager_helpers.go` — Add dispatch at activation and watch points
- `internal/session/manager_prompt.go` — Add dispatch around prompt assembly
- Event recording path — Add dispatch around event persistence

The session manager needs access to typed dispatch functions. Either inject `*hooks.Hooks` directly or define a narrow interface consumed by session that `Hooks` implements.

### Relevant Files
- `internal/session/manager_lifecycle.go` — Create (line 82), Stop (lines 317-402)
- `internal/session/manager_helpers.go` — activateAndWatch, notifier.OnSessionCreated call
- `internal/session/manager_prompt.go` — Prompt assembly path
- `internal/session/interfaces.go` — May need extended interface or Hooks injection
- `internal/hooks/hooks.go` (task_06) — Typed dispatch functions to call

### Dependent Files
- `internal/session/` — Multiple files modified with dispatch calls

### Related ADRs
- [ADR-006: Sequential Pipeline for Sync Hook Patch Composition](../adrs/adr-006.md) — Pre-create can block creation

## Deliverables
- Modified session manager files with dispatch calls at all lifecycle points
- Integration tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `session.pre_create` hook that denies — session creation returns error
  - [x] `session.pre_create` hook that patches — patched payload used for creation
  - [x] `session.post_create` async hook fires after session is active
  - [x] `input.pre_submit` hook patches message — patched message is processed
  - [x] `prompt.post_assemble` hook patches prompt — patched prompt is used
  - [x] `agent.crashed` hook fires when agent process crashes
  - [x] `event.pre_record` and `event.post_record` fire around event persistence (async-only)
- Integration tests:
  - [x] Full session lifecycle: create → input → prompt → agent events → stop — all hooks fire in order
  - [x] Permission escalation e2e: permission.request hook attempts deny→allow — blocked by invariant
  - [x] Pre-stop hook with required flag — hook failure prevents clean stop, error propagated
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Sync barriers can block operations — pre_create deny prevents session creation
- Existing session manager error handling and cleanup are preserved
- No regression in session lifecycle without hooks configured

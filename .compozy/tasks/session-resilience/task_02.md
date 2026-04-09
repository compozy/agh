---
status: pending
title: Stop classification + cause propagation
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 02: Stop classification + cause propagation

## Overview

Implement the stop reason classification logic in `finalizeStopped()` and propagate `StopCause` through all stop initiation points: `Stop()`, `handleProcessExit()`, and daemon shutdown. After this task, every session stop produces a classified `StopReason` persisted to meta.json and available in `SessionInfo`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `classifyStopReason(cause StopCause, waitErr error, detail string) (store.StopReason, string)` in `internal/session`
- MUST wire `classifyStopReason()` into `finalizeStopped()` before event recording
- MUST set `StopCause` explicitly at each stop initiation point — no `ctx.Err()` inference
- MUST propagate `CauseUserRequested` from `Manager.Stop()`
- MUST propagate `CauseShutdown` from `daemon.stopSessions()`
- MUST propagate `CauseCompleted` from `handleProcessExit()` when process exits cleanly without stop request
- MUST propagate `CauseProcessExited` from `handleProcessExit()` when process exits unexpectedly
- MUST persist classified StopReason to SessionMeta via `writeMeta()`
- MUST include StopReason in the `session_stopped` event payload
- MUST add `stopWasRequested()` or equivalent to Session for clean exit vs crash distinction
</requirements>

## Subtasks
- [ ] 2.1 Implement `classifyStopReason()` function with deterministic mapping from StopCause + waitErr
- [ ] 2.2 Wire classification into `finalizeStopped()` — set session.stopReason/stopDetail, write meta, include in stop event
- [ ] 2.3 Modify `Stop()` to set `CauseUserRequested` on the session before proceeding
- [ ] 2.4 Modify `handleProcessExit()` to set `CauseCompleted` or `CauseProcessExited` based on waitErr and stop-request state
- [ ] 2.5 Modify `daemon.stopSessions()` to set `CauseShutdown` on each session before calling Stop
- [ ] 2.6 Write unit and integration tests for classification and cause propagation

## Implementation Details

See TechSpec "Stop Reason Classification Logic" and "Stop Cause Propagation" sections for the classification switch and propagation table.

The key design principle: `StopCause` is set by the code path that initiates the stop, BEFORE `finalizeStopped()` runs. `finalizeStopped()` reads the cause and maps it deterministically. No ambiguity.

### Relevant Files
- `internal/session/manager_lifecycle.go` — `finalizeStopped()` (line 317), `Stop()` (line 128), `handleProcessExit()` (line 304), `watchProcess()` (line 285)
- `internal/session/session.go` — `Session` struct, `prepareStop()` (line 285)
- `internal/daemon/daemon.go` — `stopSessions()` (line 465), shutdown sequence (line 369)

### Dependent Files
- `internal/store/meta.go` — `WriteSessionMeta()` persists the classified reason
- `internal/observe/observer.go` — `OnSessionStopped()` will read StopReason (task 03)
- `internal/session/manager_lifecycle.go` — Resume repair will use classified StopReason (task 04)

### Related ADRs
- [ADR-001: Canonical StopReason Enum on SessionMeta](adrs/adr-001.md) — Classification uses explicit StopCause, not ctx.Err() inference

## Deliverables
- `classifyStopReason()` function in `internal/session`
- StopCause propagation in Stop(), handleProcessExit(), daemon.stopSessions()
- StopReason persisted in meta.json and session_stopped event
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for stop flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `classifyStopReason(CauseShutdown, nil, "")` → `StopShutdown`
  - [ ] `classifyStopReason(CauseShutdown, someErr, "")` → `StopShutdown` (shutdown wins)
  - [ ] `classifyStopReason(CauseUserRequested, nil, "")` → `StopUserCanceled`
  - [ ] `classifyStopReason(CauseUserRequested, nil, "max_iterations")` → `StopMaxIterations`
  - [ ] `classifyStopReason(CauseUserRequested, nil, "loop_detected")` → `StopLoopDetected`
  - [ ] `classifyStopReason(CauseUserRequested, nil, "budget_exceeded")` → `StopBudgetExceeded`
  - [ ] `classifyStopReason(CauseProcessExited, waitErr, "")` → `StopAgentCrashed`
  - [ ] `classifyStopReason(CauseProcessExited, nil, "")` → `StopError`
  - [ ] `classifyStopReason(CauseCompleted, nil, "")` → `StopCompleted`
  - [ ] `classifyStopReason(CauseHookDenied, nil, "reason")` → `StopHookStopped`
  - [ ] `classifyStopReason(CauseNone, waitErr, "")` → `StopError` (fallback)
  - [ ] `classifyStopReason(CauseNone, nil, "")` → `StopCompleted` (fallback)
- Integration tests:
  - [ ] Create session → Stop() → verify meta.json has `stop_reason: "user_canceled"`
  - [ ] Create session → kill subprocess → verify meta.json has `stop_reason: "agent_crashed"`
  - [ ] Create session → daemon shutdown → verify meta.json has `stop_reason: "shutdown"`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Every session stop path produces a classified StopReason
- StopReason persisted in meta.json for all stop scenarios

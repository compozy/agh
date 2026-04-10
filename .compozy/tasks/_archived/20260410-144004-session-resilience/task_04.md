---
status: completed
title: Resume repair + config + verification
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 04: Resume repair + config + verification

## Overview

Implement the infrastructure-level repair pipeline in `Resume()` that validates session state before starting the ACP agent. Add `SessionLimitsConfig` with a `timeout` field. Write end-to-end integration tests verifying the complete stop reason + resume repair flow. This is the final task that ties all pieces together and ensures the system works end-to-end.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `classifyPreviousStop(meta)` that detects crashed sessions from meta state
- MUST implement `validateInfrastructure(meta)` returning independent errors per check
- MUST check: workspace dir exists, agent definition resolvable, event store file exists and non-zero, meta fields valid
- MUST insert repair pipeline into `Resume()` after `ReadSessionMeta()` and before `resolveResumeWorkspace()`
- MUST persist crash classification to meta.json (update StopReason/StopDetail)
- MUST prepare hook seams for `session.pre_resume` and `session.post_resume` as no-op function calls
- MUST add `SessionLimitsConfig` struct with `Timeout` field to `internal/config`
- MUST add TOML parsing and merge logic for `[session.limits]` config section
- MUST write end-to-end integration tests covering the full create → stop → resume flow
- SHOULD log structured events for crash classification and validation failures
</requirements>

## Subtasks
- [x] 4.1 Implement `classifyPreviousStop(meta)` — map meta.State to StopReason for crashed sessions
- [x] 4.2 Implement `validateInfrastructure(meta)` — 4 independent checks returning []error
- [x] 4.3 Insert repair pipeline into `Resume()` flow with crash classification and meta persistence
- [x] 4.4 Add hook seams as no-op functions (prepared for future `session.pre_resume`/`session.post_resume`)
- [x] 4.5 Add `SessionLimitsConfig` with `Timeout` to config, TOML parsing, and merge logic
- [x] 4.6 Write end-to-end integration tests for stop reason + resume repair flows
- [x] 4.7 Run `make verify` and fix any issues

## Implementation Details

See TechSpec "Resume Repair Pipeline" section for the step-by-step validation flow and "Phase 2: Loop/Recursion Guards (Deferred)" for the hook seam design.

The repair pipeline runs BEFORE the ACP agent starts. Each infrastructure check is independent — one failure does not block others from running. All errors are collected and returned as a combined diagnostic.

Hook seams are plain function calls that do nothing in Phase 1. When the hooks platform is ready, they become typed dispatch calls. This avoids coupling Phase 1 to hooks availability.

### Relevant Files
- `internal/session/manager_lifecycle.go` — `Resume()` (line 170), where the repair pipeline inserts
- `internal/store/meta.go` — `ReadSessionMeta()`, `WriteSessionMeta()` for crash classification persistence
- `internal/config/config.go` — existing config structs (LimitsConfig at line 41)
- `internal/config/merge.go` — overlay merge patterns for new config section

### Dependent Files
- `internal/session/manager_lifecycle.go` — Resume() modified to include repair pipeline
- `internal/config/config.go` — new SessionLimitsConfig struct
- `internal/config/merge.go` — new merge logic for session limits

### Related ADRs
- [ADR-003: Infrastructure-Level Repair on Resume](adrs/adr-003.md) — Scope of repair checks, hook seam design
- [ADR-005: Defer Loop Guards to Phase 2](adrs/adr-005.md) — Hook seams prepared but not wired

## Deliverables
- `classifyPreviousStop()` and `validateInfrastructure()` functions
- Resume repair pipeline integrated into `Resume()`
- Hook seams for future session.pre_resume/post_resume
- `SessionLimitsConfig` with TOML parsing and merge
- Unit tests with 80%+ coverage **(REQUIRED)**
- End-to-end integration tests **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `classifyPreviousStop` with meta.State="active" → StopReason="agent_crashed"
  - [x] `classifyPreviousStop` with meta.State="stopping" → StopReason="agent_crashed"
  - [x] `classifyPreviousStop` with meta.State="starting" → StopReason="error"
  - [x] `classifyPreviousStop` with meta.State="stopped" + existing StopReason → preserved
  - [x] `classifyPreviousStop` with meta.State="stopped" + nil StopReason → no change
  - [x] `validateInfrastructure` with valid workspace/agent/store/meta → no errors
  - [x] `validateInfrastructure` with missing workspace dir → error with path
  - [x] `validateInfrastructure` with unresolvable agent → error with agent name
  - [x] `validateInfrastructure` with missing event store → error with DB path
  - [x] `validateInfrastructure` with zero-size event store → error
  - [x] `validateInfrastructure` with empty meta.ID → error
  - [x] `validateInfrastructure` with multiple failures → all errors collected
  - [x] SessionLimitsConfig TOML parsing with valid timeout
  - [x] SessionLimitsConfig merge with overlay
- Integration tests:
  - [x] Create → explicit Stop → verify StopReason="user_canceled" in meta + global DB + API
  - [x] Create → kill subprocess → verify StopReason="agent_crashed" in meta + global DB + API
  - [x] Create → write meta State="active" (simulate crash) → Resume → verify crash classified
  - [x] Create → delete workspace dir → Resume → verify descriptive error
  - [x] Create → remove agent from config → Resume → verify descriptive error
  - [x] Create → truncate event store → Resume → verify descriptive error
  - [x] Create → crash → Resume → verify session activates successfully after classification
  - [x] Full flow: create → stop → resume → stop → verify both stops have correct StopReasons
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- [x] All tests passing
- [x] Test coverage >=80%
- [x] `make verify` passes
- [x] Crashed sessions are correctly classified on resume
- [x] Infrastructure validation catches all 4 failure modes with descriptive errors
- [x] Full create → stop → resume flow works end-to-end with correct StopReasons throughout

---
status: completed
title: Extract canonical transcript assembly into internal/transcript
type: refactor
complexity: medium
dependencies:
  - task_06
---

# Task 07: Extract canonical transcript assembly into internal/transcript

## Overview
This task moves replay-oriented transcript assembly out of `session` into `internal/transcript`. It clarifies that transcript rendering is a dedicated concern layered on top of persisted session events, not part of core session lifecycle ownership.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- Canonical transcript assembly MUST move from `internal/session/transcript.go` into `internal/transcript`.
- The session manager MUST keep exposing transcript behavior to callers without changing endpoint semantics.
- Transcript message ordering, role mapping, tool call/result handling, and thinking content behavior MUST remain stable.
- Replay-specific tests MUST move with the new package ownership rather than leaving the old package as the long-term owner.
</requirements>

## Subtasks
- [x] 7.1 Create `internal/transcript` and move canonical transcript assembly into it.
- [x] 7.2 Update `session` to delegate transcript assembly instead of owning replay logic directly.
- [x] 7.3 Migrate or rewrite transcript-focused tests under the new package ownership.
- [x] 7.4 Remove obsolete replay-specific code from `session` after delegation is stable.

## Implementation Details
Use the TechSpec `Component Overview`, `Core Interfaces`, and `Data Models` sections. This is a package-ownership move, not a transcript format redesign. Preserve the canonical replay response expected by the transcript endpoint.

### Relevant Files
- `internal/session/transcript.go` — Current home of canonical replay assembly.
- `internal/session/transcript_test.go` — Existing replay behavior coverage that should move with the new package.
- `internal/session/manager.go` — Calls transcript assembly through the manager surface.
- `internal/api/core/handlers.go` or its migrated equivalent — Consumes transcript output for API responses.
- `internal/store` or `internal/store/sessiondb` event query surfaces — Provide the ordered events transcript assembly depends on.

### Dependent Files
- `internal/session/manager_test.go` — May need updated expectations around transcript delegation.
- `internal/httpapi/handlers_test.go` or migrated API tests — Must continue validating transcript endpoint behavior.
- `internal/udsapi/handlers_test.go` or migrated API tests — Must continue validating transcript endpoint behavior.

### Related ADRs
- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](../adrs/adr-001.md) — Establishes transcript extraction as part of the target architecture.

## Deliverables
- `internal/transcript` package owning canonical transcript assembly.
- `session` package updated to delegate transcript rendering.
- Transcript unit and endpoint-adjacent tests updated with at least 80% coverage for touched code.
- Stable transcript endpoint behavior after the move.

## Tests
- Unit tests:
  - [x] User, assistant, thought, tool-call, and tool-result events still assemble into the same canonical transcript sequence.
  - [x] Mixed turn ordering still produces stable ordering by sequence and timestamp.
  - [x] Empty or ignorable events still behave the same as before in transcript output.
- Integration tests:
  - [x] Session transcript endpoint still returns the same replay structure through the API after the extraction.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Transcript replay ownership moves out of `session`
- API transcript behavior remains unchanged

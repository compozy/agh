---
status: pending
title: Integrate turn, message, and context dispatch
type: backend
complexity: medium
dependencies:
  - task_10
---

# Task 11: Integrate turn, message, and context dispatch

## Overview

Wire typed dispatch calls for `turn.*`, `message.*`, and `context.*` hook families into the ACP event flow and compaction paths. Turn and message hooks observe the normalized ACP event stream. Context hooks wrap the compaction/context-window management operations.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `turn.start` and `turn.end` dispatch at turn boundaries in the ACP event flow
- MUST add `message.start` sync dispatch at message begin
- MUST add `message.delta` async-only dispatch for streaming tokens
- MUST add `message.end` sync dispatch at message completion
- MUST add `context.pre_compact` sync dispatch before compaction — can patch compaction params
- MUST add `context.post_compact` async dispatch after compaction completes
- MUST ensure `message.delta` dispatch does not block token streaming (async-only enforced by event eligibility)
- MUST ensure `turn.start`/`turn.end` fire at correct ACP event boundaries
</requirements>

## Subtasks
- [ ] 11.1 Add turn.start and turn.end dispatch in ACP event processing
- [ ] 11.2 Add message.start, message.delta, and message.end dispatch in message flow
- [ ] 11.3 Add context.pre_compact and context.post_compact dispatch around compaction
- [ ] 11.4 Verify message.delta is async-only and does not block streaming
- [ ] 11.5 Write unit tests for turn, message, and context hooks

## Implementation Details

Modify existing files in:
- ACP event processing path — Where agent events are normalized and dispatched
- Compaction/context management path — Where context window compaction happens
- Session event flow — Where turn and message boundaries are identified

### Relevant Files
- `internal/session/` — ACP event processing and turn management
- `internal/acp/` — Agent event types and turn boundaries
- `internal/hooks/hooks.go` (task_06) — Typed dispatch functions
- `internal/hooks/events.go` (task_01) — message.delta is async-only

### Dependent Files
- ACP event flow files — Modified with dispatch calls

### Related ADRs
- [ADR-012: Classify Events into Sync-Eligible and Async-Only](../adrs/adr-012.md) — message.delta is async-only

## Deliverables
- Modified ACP event flow with turn and message dispatch
- Modified compaction path with context dispatch
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `turn.start` fires when a new turn begins in ACP event flow
  - [ ] `turn.end` fires when a turn completes
  - [ ] `message.start` sync hook fires at message begin
  - [ ] `message.delta` dispatches asynchronously — does not block token delivery
  - [ ] `message.end` sync hook fires at message completion
  - [ ] `context.pre_compact` hook patches compaction params — patched params used
  - [ ] `context.post_compact` fires after compaction completes
- Integration tests:
  - [ ] Full message flow: start → deltas → end — hooks fire at correct points
  - [ ] Compaction with pre_compact hook — verifies patched params are used
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Token streaming is not blocked by message.delta hooks
- Turn boundaries correctly identified in ACP event flow

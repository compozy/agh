---
status: completed
title: Synthetic prompt submission and session-event persistence model
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 04: Synthetic prompt submission and session-event persistence model

## Overview

Create the dedicated synthetic-turn path for harness-driven prompt submission and persistence. This task introduces a real internal origin and event model for daemon-generated prompt turns, instead of reusing `user_message` semantics that would break transcript trust boundaries and downstream consumers.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_01.md`, and the current prompt persistence path before starting
- REFERENCE TECHSPEC sections "Workstream 4: Synthetic Reentry Model" and "Required Behavior"
- FOCUS ON "WHAT" - add a first-class synthetic turn and event model; do not yet wire detached task-run completion into it here
- MINIMIZE CODE - extend the existing prompt submission path carefully rather than creating a second unrelated prompt API
- TESTS REQUIRED - new turn origin, persisted event type, and queue semantics need direct coverage
- GREENFIELD: reentry sintetica nao pode cair em `EventTypeUserMessage` nem mascarar origem humana
</critical>

<requirements>
- MUST add a dedicated synthetic turn origin and a dedicated persisted event type for daemon-originated prompt submission
- MUST keep synthetic prompt submission daemon-owned and unavailable as a normal user-facing transport path
- MUST avoid persisting synthetic turns as `EventTypeUserMessage`
- MUST carry enough metadata for later detached-run reentry to target one task run or wake-up reason explicitly
- SHOULD preserve compatibility for existing user and network prompt submission paths
</requirements>

## Subtasks
- [x] 4.1 Extend the turn-origin vocabulary and prompt submission path to support synthetic turns
- [x] 4.2 Add a dedicated persisted event type and payload for synthetic prompt input
- [x] 4.3 Thread synthetic metadata through the canonical prompt request normalization path
- [x] 4.4 Define queue and ordering behavior for synthetic turns when a session is already busy
- [x] 4.5 Add tests covering validation, persistence, and ordering semantics

## Implementation Details

See TechSpec "Workstream 4: Synthetic Reentry Model" and ADR-001/ADR-003. This task is the trust-boundary fix that allows later background completion or daemon-internal wake-ups to exist without pretending they came from a human or external network peer.

### Relevant Files
- `internal/session/interfaces.go` - current turn-origin model must expand to represent synthetic turns explicitly
- `internal/session/manager_prompt.go` - current prompt submission and input persistence path must gain a synthetic branch
- `internal/acp/types.go` - event-type vocabulary may need extension or adjacent constants for the new persisted semantics
- `internal/store/types.go` - per-session event shape and validation must support the new synthetic event type
- `internal/store/sessiondb/session_db.go` - persistent event storage and query paths must continue to accept and order the new event type correctly
- `internal/session/synthetic_prompt.go` - new daemon-owned helper path for synthetic prompt submission introduced by this task

### Dependent Files
- `internal/transcript/transcript.go` - later transcript behavior will consume the new synthetic event type
- `internal/session/manager_hooks.go` - hook input classification must later distinguish synthetic reentry from user/network turns
- `internal/extension/host_api.go` - extension prompt submission assumptions about `user_message` will later depend on this new model
- `internal/session/manager_test.go` - prompt and persistence tests must gain synthetic-turn coverage

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - Adds the synthetic turn-origin axis cleanly
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - Defines why this synthetic path is needed for later detached-run wake-up behavior

### External References
- `.resources/claude-code/utils/messages.ts` - strong reference for representing synthetic/internal message origins distinctly from user content
- `.resources/claude-code/utils/sdkEventQueue.ts` - useful model for internal event classes attached to background and notification flows
- `.resources/claude-code/query.ts` - demonstrates queue draining that distinguishes prompt work from task notifications
- `.resources/hermes/gateway/run.py` - useful precedent for daemon-internal `internal=True` message injection
- `.resources/openfang/crates/openfang-types/src/webhook.rs` - good reference for structured wake-up payloads and internal delivery metadata
- `.resources/openfang/crates/openfang-runtime/src/agent_loop.rs` - helpful reference for turn lifecycle around internally injected work

## Deliverables
- Synthetic turn origin and daemon-owned prompt submission path
- Dedicated persisted event type for synthetic prompt input **(REQUIRED)**
- Validation and ordering rules for synthetic prompt submission **(REQUIRED)**
- No persistence path that reuses `EventTypeUserMessage` for daemon-originated turns **(REQUIRED)**
- Queueing and metadata regression coverage for synthetic prompt submission against active sessions **(REQUIRED)**
- Unit and integration tests with >=80% coverage for the new synthetic prompt path **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Synthetic prompt requests normalize successfully only through the daemon-owned submission path and reject ordinary user-facing callers
  - [x] Synthetic prompt input persists with the dedicated event type and payload envelope instead of `user_message`
  - [x] Invalid synthetic payloads, missing wake-up metadata, or unsupported synthetic origins fail validation clearly
  - [x] Synthetic turns carry task-run id, wake-up reason, and summary metadata when provided by the caller
  - [x] Session-busy queue rules preserve the synthetic turn until it can execute instead of dropping it silently
- Integration tests:
  - [x] A synthetic prompt can be recorded and dispatched through the normal manager pipeline without breaking ordinary user and network prompt paths
  - [x] Busy-session queue behavior preserves ordering when a synthetic wake-up is submitted behind an active user or network turn
  - [x] Persisted event history for mixed user plus synthetic turns remains queryable through the standard session event APIs
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- AGH can represent daemon-originated prompt turns without violating transcript and audit trust boundaries
- The runtime is ready for detached task-run wake-ups to target a real synthetic input path

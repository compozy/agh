---
status: pending
title: Inbound delivery workers and turn-end handoff
type: backend
complexity: high
dependencies:
  - task_03
  - task_05
---

# Task 06: Inbound delivery workers and turn-end handoff

## Overview

Create the inbound delivery layer that queues routed messages per session and delivers them only when the target session is ready. This task replaces any global delivery-loop concept with per-session workers and a safe message wrapper for prompt injection resistance.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST maintain a bounded inbound queue per session with FIFO semantics and overflow behavior from the tech spec
- MUST deliver inbound network messages through per-session workers rather than a single daemon-global loop
- MUST integrate with turn-end notifications so delivery occurs only when the target session is ready for another prompt
- MUST render inbound messages with the safe wrapper format defined by the corrected tech spec
</requirements>

## Subtasks
- [ ] 6.1 Implement per-session inbox queues and worker lifecycle management under `internal/network`
- [ ] 6.2 Connect delivery triggering to turn-end notifications and session readiness
- [ ] 6.3 Build safe inbound message rendering with escaped preview and structured body payload
- [ ] 6.4 Add concurrency and overflow tests for delivery ordering and shutdown behavior

## Implementation Details

This task should consume the routing outputs from task 03 and the prompt provenance surface from task 05. Keep ownership of delivery concurrency inside `internal/network`.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Inbound delivery, prompt wrapper, queue depth, and build-order sections
- `internal/network/delivery.go` - New inbox queue and worker implementation
- `internal/network/router.go` - Routed inbound envelopes arrive here before delivery
- `internal/session/interfaces.go` - Delivery workers need a session-facing prompt and turn-end surface
- `internal/daemon/hooks_bridge.go` - Existing notifier patterns inform turn-end integration

### Dependent Files
- `internal/network/manager.go` - Manager will own worker lifecycle and session registration
- `internal/daemon/boot.go` - Daemon boot will wire delivery services into runtime deps later
- `internal/api/contract/contract.go` - Inbox/status APIs will eventually surface queue state

### Related ADRs
- [ADR-003: CLI + Bundled Skill for Agent Network Communication](adrs/adr-003.md) - Inbound path is queue-and-deliver-after-turn
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - Delivery remains daemon-owned infrastructure instead of session-owned network code

## Deliverables
- Delivery worker and inbox queue implementation under `internal/network`
- Safe wrapper rendering for inbound messages
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for turn-end delivery sequencing **(REQUIRED)**

## Tests
- Unit tests:
- [ ] Queues preserve FIFO ordering per session and drop oldest messages on configured overflow
- [ ] Worker lifecycle starts and stops cleanly without leaking goroutines
- [ ] Safe wrapper rendering escapes untrusted preview content and preserves the structured payload body
- [ ] Idle and busy session states produce the expected immediate versus deferred delivery behavior
- Integration tests:
- [ ] Concurrent inbound messages for multiple sessions are delivered independently without cross-session head-of-line blocking
- [ ] A busy session receives exactly one queued network prompt after each turn ends until the queue drains
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Inbound delivery is safe, bounded, and per-session
- The runtime has a reliable turn-end handoff path for network messages

---
status: completed
title: CLI Commands — Messaging & State
type: ""
complexity: medium
dependencies:
    - task_04
    - task_05
    - task_10
    - task_17
---

# Task 18: CLI Commands — Messaging & State

## Overview
Implement the Cobra CLI subcommands for agent messaging (send, broadcast, escalate) and state management (state read, state append, context, status, events). These commands enable inter-agent communication and shared state access.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement agh send with agent-id and message arguments per docs/spec-v2/06-cli.md
- MUST implement agh broadcast with message argument
- MUST implement agh escalate with message argument (master-only)
- MUST implement agh state read with --type, --scope, --limit, --since flags
- MUST implement agh state append with --type flag and content argument
- MUST implement agh context (aggregated TOON view)
- MUST implement agh status for both update (own status) and read (other agent) modes
- MUST implement agh events with --type, --limit, --scope, --agent flags
- MUST render all output in TOON format
- MUST enforce scope rules on send (same workgroup or parent)
</requirements>

## Subtasks
- [x] 18.1 Implement messaging commands: send, broadcast, escalate
- [x] 18.2 Implement state read with filtering (type, scope, limit, since)
- [x] 18.3 Implement state append with type classification
- [x] 18.4 Implement context command (aggregated workgroup view)
- [x] 18.5 Implement status command (dual mode: update own, read others)
- [x] 18.6 Implement events command with filtering

## Implementation Details
Refer to docs/spec-v2/06-cli.md for all command specs and output format examples.

### Relevant Files
- `docs/spec-v2/06-cli.md` — CLI reference for messaging and state commands
- `docs/spec-v2/04-workgroups.md` — scoping rules for messaging

### Dependent Files
- `internal/cli/root.go` — HTTP-over-UDS connection helper from task_17 (~/.agh/daemon.sock)
- `internal/toon/renderer.go` — TOON rendering from task_10
- `internal/transport/` — NATS messaging

## Deliverables
- internal/cli/messaging.go — send, broadcast, escalate commands
- internal/cli/state.go — state read, state append, context, status, events commands
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Send builds correct NATS message with target agent and content
  - [x] Broadcast publishes to workgroup broadcast subject
  - [x] Escalate publishes to parent workgroup escalate subject
  - [x] State read applies filters: type, scope, limit, since
  - [x] State append creates blackboard entry with correct type
  - [x] Context renders aggregated TOON (workgroup + agents + blackboard)
  - [x] Status update mode vs read mode dispatched correctly
  - [x] Events filters by type, agent, scope, limit
  - [x] TOON output matches spec examples
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All commands match docs/spec-v2/06-cli.md spec

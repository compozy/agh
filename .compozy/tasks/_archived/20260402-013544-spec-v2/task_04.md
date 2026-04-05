---
status: completed
title: NATS Embedded Transport
type: ""
complexity: medium
dependencies:
    - task_01
---

# Task 4: NATS Embedded Transport

## Overview
Implement the embedded NATS message broker with in-process operation (no TCP port), the full subject hierarchy for workgroup-scoped messaging, scope enforcement that rejects cross-workgroup publishes, and a Unix Domain Socket bridge for CLI-to-kernel communication.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>

> **Note:** The original implementation used per-session sockets. The daemon architecture uses a single global socket (`~/.agh/daemon.sock`) for multi-session support with HTTP over UDS (Gin).

- MUST embed NATS server with DontListen: true (no TCP port) per docs/spec-v2/02-kernel.md
- MUST implement the full subject hierarchy: agh.wg.{xid}.agent.{xid}, agh.wg.{xid}.broadcast, agh.wg.{xid}.hook, agh.wg.{xid}.blackboard, agh.wg.{xid}.status, agh.wg.{xid}.escalate, agh.system.ready.{xid}, agh.system.health.{xid}
- MUST enforce scoping rules: within-workgroup allowed, cross-workgroup blocked, master-only escalation, parent-to-child allowed
- MUST implement UDS bridge at ~/.agh/daemon.sock for CLI connections (HTTP over UDS with Gin)
- MUST use request-reply pattern for CLI commands (publish request, wait for response)
- MUST return clear error messages on scope violations per docs/spec-v2/04-workgroups.md
- MUST support nc.Drain() for graceful NATS shutdown

</requirements>

## Subtasks
- [x] 4.1 Set up embedded NATS server with DontListen: true and in-process client connection
- [x] 4.2 Implement subject hierarchy with workgroup and system subjects
- [x] 4.3 Implement scope enforcement middleware (validate publisher workgroup membership)
- [x] 4.4 Implement UDS listener that bridges incoming connections to embedded NATS
- [x] 4.5 Implement request-reply pattern for CLI command handling
- [x] 4.6 Implement graceful NATS shutdown with nc.Drain()

## Implementation Details
Refer to docs/spec-v2/02-kernel.md for NATS setup and subject hierarchy. Refer to docs/spec-v2/04-workgroups.md for scoping rules and error messages.

### Relevant Files
- `docs/spec-v2/02-kernel.md` — NATS setup, subject hierarchy, UDS bridge
- `docs/spec-v2/04-workgroups.md` — scoping rules with examples
- `docs/spec-v2/08-data-models.md` — NATS subject catalog

### Dependent Files
- `internal/kernel/types.go` — Message struct
- `internal/registry/` — agent workgroup membership lookups for scope validation

## Deliverables
- internal/transport/nats.go — embedded NATS server and client
- internal/transport/uds.go — Unix Domain Socket bridge
- internal/transport/scoping.go — scope enforcement logic
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for scoped messaging **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Embedded NATS starts and accepts in-process connections
  - [x] Subject hierarchy correctly routes messages to subscribers
  - [x] Same-workgroup message delivery succeeds
  - [x] Cross-workgroup message is rejected with scope violation error
  - [x] Broadcast reaches all agents in workgroup, none outside
  - [x] Master can escalate to parent workgroup
  - [x] Non-master escalation is rejected with clear error message
  - [x] Parent workgroup master can publish to child workgroup subjects
- Integration tests:
  - [x] UDS listener at ~/.agh/daemon.sock accepts CLI connections and bridges to NATS
  - [x] Request-reply pattern returns responses to CLI caller
  - [x] nc.Drain() completes in-flight handlers before shutdown
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- No TCP ports opened (DontListen: true verified)
- Scope violation errors match messages from docs/spec-v2/04-workgroups.md

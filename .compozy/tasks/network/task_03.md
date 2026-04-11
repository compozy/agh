---
status: completed
title: Presence registry and router
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Presence registry and router

## Overview

Implement runtime peer tracking and message routing on top of the protocol and transport foundations. This task is responsible for local and remote peer visibility, heartbeat-driven freshness, deduplication, and sender-side preflight for directed sends.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST maintain local peer membership by session and remote peer cache keyed by `peer_id`, scoped by space
- MUST implement `greet`, `leave`, heartbeat freshness, and `whois` handling required for discovery and directed routing
- MUST enforce sender-side presence preflight for directed sends so absent or expired targets fail locally without publish
- MUST route broadcast and direct traffic with deduplication, receiver ordering, and lifecycle-aware rejection behavior from the tech spec
</requirements>

## Subtasks
- [x] 3.1 Implement peer registry and remote peer cache structures under `internal/network`
- [x] 3.2 Add heartbeat, greet, leave, and whois handling with expiry semantics
- [x] 3.3 Build router logic for subject mapping, deduplication, direct versus broadcast routing, and local preflight
- [x] 3.4 Cover presence and routing policy with unit and integration tests

## Implementation Details

This task should continue to keep all protocol and transport behavior inside `internal/network`. Session participation and daemon boot wiring belong to later tasks.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Presence heartbeat, receiver rules, subject mapping, and build-order sections
- `internal/network/peer.go` - New peer registry and remote cache implementation
- `internal/network/router.go` - New routing, deduplication, and sender preflight logic
- `internal/network/transport.go` - Existing transport surface from task 02
- `docs/rfcs/003_agh-network-v0.md` - RFC guidance for discovery, routing, and lifecycle ordering

### Dependent Files
- `internal/network/delivery.go` - Delivery workers depend on routed inbound envelopes
- `internal/network/manager.go` - Manager will delegate presence and routing responsibilities here
- `internal/api/contract/contract.go` - Network send and peers/status APIs will surface registry and routing outcomes
- `internal/cli/network.go` - CLI commands will eventually consume peer and routing results

### Related ADRs
- [ADR-001: Embedded NATS Server as Transport Layer](adrs/adr-001.md) - Router consumes the daemon-owned transport boundary
- [ADR-002: Session-as-Peer Identity Model](adrs/adr-002.md) - Peer registry must respect session-scoped identity
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Presence is runtime-only and space-scoped

## Deliverables
- Peer registry and router implementations under `internal/network`
- Sender-side directed-send preflight and deduplication behavior aligned with the tech spec
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for discovery, heartbeat freshness, and routing behavior **(REQUIRED)**

## Tests
- Unit tests:
- [x] Local and remote peers are isolated by space and expire on schedule
- [x] Directed sends to absent or expired peers fail locally without publish
- [x] Broadcast routing and direct routing choose the correct subjects and targets
- [x] Duplicate envelopes are rejected without reprocessing lifecycle state
- Integration tests:
- [x] Two in-process peers can greet, observe each other, and exchange direct and broadcast messages through the router
- [x] Stale peers disappear after heartbeat expiry and recover after fresh greet traffic
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Presence and routing behavior match the corrected tech spec
- Later delivery and daemon tasks can rely on stable registry and router interfaces

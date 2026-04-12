---
status: completed
title: Build delivery broker and session-to-channel projection
type: backend
complexity: critical
dependencies:
  - task_02
  - task_03
  - task_04
---

# Task 06: Build delivery broker and session-to-channel projection

## Overview

Build the daemon-side delivery broker that converts session output into a delivery-oriented stream for channel adapters. This task owns the most concurrency-sensitive part of the feature: ordered delivery by routing key, bounded queues, ack handling, delta coalescing, and resumable delivery snapshots.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a `DeliveryBroker` in `internal/channels/` that projects session output into delivery-oriented events instead of pushing raw ACP events to extensions.
2. MUST guarantee ordering per routing key and use bounded queues per channel instance and routing key, with coalescing behavior that never drops `start`, `final`, or `error` events.
3. MUST support request/response acknowledgements, progressive remote message IDs, and resumable delivery snapshots for extension reconnect or restart scenarios.
4. SHOULD keep hooks and observability concerns separate from delivery transport so channel streaming does not become a second hook system.
</requirements>

## Subtasks
- [x] 6.1 Implement the core delivery broker, queueing model, and delivery-event lifecycle
- [x] 6.2 Project session output into delivery events using the existing session notifier/prompt boundaries
- [x] 6.3 Add ack, replacement ID, and resumable snapshot handling for active deliveries
- [x] 6.4 Add concurrency, ordering, and recovery tests for the broker

## Implementation Details

Follow the TechSpec sections "Outbound", "Semantics of the stream", "Session Manager", and "Testing Approach". This task should focus on delivery runtime behavior only; it should not expose HTTP/CLI transports or create a reference adapter.

### Relevant Files
- `internal/session/interfaces.go` — The session notifier and prompt boundaries define the existing seam for session output projection
- `internal/session/manager_prompt.go` — Prompt execution currently emits agent events and is the most likely integration point for delivery projection
- `internal/hooks/events.go` — Hook taxonomy must stay separate from the new delivery stream semantics
- `internal/extension/manager.go` — Outbound delivery requests will eventually flow through the negotiated extension runtime managed here

### Dependent Files
- `internal/daemon/boot.go` — Daemon composition later needs to wire the delivery broker into channel runtime services
- `internal/observe/health.go` — Later observability work depends on broker backlog and delivery-state reporting
- `internal/extension/manager_integration_test.go` — Integration coverage later needs to exercise negotiated delivery behavior end to end

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Confirms the delivery runtime belongs in the daemon substrate
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — Defines the delivery-oriented stream semantics and recovery requirements

## Deliverables
- A daemon-side delivery broker with ordered per-route delivery behavior
- Session-output projection, ack handling, and resumable snapshot support
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for ordered delivery and restart recovery **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Delivery events for the same routing key are emitted in order even when a second routing key is active concurrently
  - [x] Backpressure coalesces intermediate delta events while preserving `start`, `final`, and `error`
  - [x] Ack handling records `remote_message_id` and replacement IDs without losing the original delivery association
  - [x] Snapshot generation for an active delivery captures enough state to resume after an extension restart
- Integration tests:
  - [x] One prompted session produces an ordered delivery stream consumed by a fake channel extension over the negotiated channel service
  - [x] A slow fake adapter triggers bounded-queue behavior without dropping terminal delivery events
  - [x] Restarting the fake adapter causes the broker to resume or fail the active delivery explicitly rather than silently losing it
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Session output is delivered as a channel-specific stream with explicit ordering and backpressure semantics
- Extension restarts do not silently orphan active outbound deliveries

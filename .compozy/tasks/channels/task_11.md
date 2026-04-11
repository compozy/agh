---
status: completed
title: Implement the Telegram reference adapter and adapter conformance harness
type: backend
complexity: critical
dependencies:
  - task_05
  - task_06
  - task_08
  - task_10
---

# Task 11: Implement the Telegram reference adapter and adapter conformance harness

## Overview

Implement a reference Telegram adapter extension and the conformance harness that proves channel adapters can use the approved runtime correctly. This task should demonstrate the full inbound/outbound contract, per-instance launch binding, negotiated delivery, health reporting, and adapter ack semantics without leaving the daemon to guess how a real adapter should behave.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a reference Telegram adapter extension that uses `channels/messages/ingest` for inbound traffic and the negotiated `channels/deliver` service for outbound delivery.
2. MUST demonstrate instance-scoped launch metadata and bound credential injection without requiring arbitrary runtime secret reads.
3. MUST add an adapter conformance harness that verifies ingest, delivery ordering, ack semantics, health reporting, and restart behavior for any future channel adapter.
4. SHOULD place the reference adapter in a repo location consistent with existing extension examples, rather than inside the daemon binary.
</requirements>

## Subtasks
- [x] 11.1 Add a reference Telegram adapter extension scaffold in the SDK/examples area of the repo
- [x] 11.2 Implement the adapter's inbound ingest, outbound delivery, ack, and health-reporting behavior against the new contracts
- [x] 11.3 Add a reusable conformance harness for channel adapters using subprocess-backed tests
- [x] 11.4 Add integration tests for adapter startup, bound credentials, delivery ordering, and restart recovery

## Implementation Details

Follow the TechSpec sections "System Architecture", "Outbound", "Recovery", and "Testing Approach". The reference adapter should validate the substrate and protocol, not expand the daemon with Telegram-specific SDK logic. Keep CI-safe tests focused on subprocess harnesses and fake platform edges rather than requiring live Telegram credentials or network calls.

### Relevant Files
- `sdk/examples/secret-guard/extension.toml` — Existing example extension layout is the best reference for where a channel adapter example should live
- `sdk/examples/prompt-enhancer/README.md` — Existing SDK example documentation pattern should be mirrored for the reference adapter
- `internal/extension/reference_integration_test.go` — Existing extension reference integration tests are the closest pattern for subprocess-backed adapter coverage
- `internal/extension/reference_support_test.go` — Helper coverage for extension reference behavior should inform the conformance harness design
- `internal/extension/manager_integration_test.go` — End-to-end runtime negotiation tests should expand for the Telegram reference adapter

### Dependent Files
- `internal/extension/manager.go` — The adapter depends on the negotiated channel-delivery runtime and launch-time binding behavior
- `internal/extension/host_api.go` — The adapter's inbound and state-reporting behavior depends on the new channel Host API methods
- `internal/observe/health.go` — Conformance tests should verify that adapter health and status changes reach the per-instance observability surface

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Confirms platform adapters remain subprocess extensions
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — The reference adapter must rely on daemon-owned routing, not local session maps
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — The reference adapter and harness must prove negotiated delivery semantics
- [ADR-008: Bound Secret Injection per Channel Instance](adrs/adr-008.md) — The reference adapter must prove instance-scoped credential binding rather than arbitrary secret reads

## Deliverables
- Reference Telegram adapter extension under the repo's extension-example area
- Reusable adapter conformance harness for subprocess-backed channel adapters
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for adapter negotiation, ingress, delivery, and restart behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] The Telegram reference adapter maps one inbound platform message into the normalized `InboundMessageEnvelope` shape expected by the Host API
  - [x] The adapter's outbound delivery handler preserves ordered `seq` handling and ack metadata for progressive edits
  - [x] The adapter reads only the bound launch-time credentials for its channel instance and does not depend on arbitrary runtime secret lookup
  - [x] The conformance harness flags missing ack, out-of-order delivery, or missing health reporting as adapter failures
- Integration tests:
  - [x] Starting the Telegram reference adapter as a subprocess negotiates `channels/deliver` and receives the expected channel instance metadata
  - [x] A fake inbound Telegram update flows through `channels/messages/ingest`, resolves a route, and results in one outbound delivery stream
  - [x] Restarting the reference adapter during an active delivery exercises broker snapshot or explicit failure behavior rather than silently losing the delivery
  - [x] Per-instance health and status changes emitted by the adapter appear in the daemon observability surface
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The repo includes one working reference adapter that proves the channel substrate end to end
- Future channel adapters can be validated against the same conformance harness instead of inventing their own runtime behavior

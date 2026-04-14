---
status: pending
title: "Integrate network ingress and channel binding for tasks"
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
---

# Task 12: Integrate network ingress and channel binding for tasks

## Overview
Integrate the task domain with network peers and channel-oriented routing so tasks can be created, filtered, and audited by operational lane. This task should make network ingress a first-class writer path while preserving idempotency, trusted origin derivation, and the revised channel-binding policy from the TechSpec.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. Network peers MUST be able to create and manipulate tasks and runs only through validated manager calls that preserve server-derived identity and immutable origin metadata.
2. Channel-bound tasks MUST validate `network_channel` consistently, reject ingress-channel mismatches, and apply the stale-channel policy accepted in the revised spec and ADRs.
3. Network-originated writes MUST preserve idempotency and auditability across retries and peer-delivery boundaries.
</requirements>

## Subtasks
- [ ] 12.1 Define the network-to-task integration seam for create/update/run operations.
- [ ] 12.2 Validate network channel binding and mismatch handling for task-backed ingress.
- [ ] 12.3 Carry peer identity, origin, and idempotency metadata into task manager calls.
- [ ] 12.4 Apply stale-channel behavior for task records and run snapshots as defined in the revised spec.
- [ ] 12.5 Add audit coverage for network-originated task writes and rejections.

## Implementation Details
Use the TechSpec sections "Authorization Contract", "API Surface", "Integration Points", and "Known Risks" plus ADR-004 for channel semantics. Follow the existing patterns in the `internal/network` package for peer validation, delivery, routing, and auditing.

### Relevant Files
- `internal/network/manager.go` — Primary network runtime entrypoint and composition surface.
- `internal/network/router.go` — Existing routing behavior that task-backed network messages must integrate with.
- `internal/network/delivery.go` — Existing delivery and retry path relevant to task ingress and idempotency.
- `internal/network/validate.go` — Existing validation patterns for network-facing inputs.
- `internal/network/audit.go` — Existing audit behavior that task ingress should extend.
- `internal/store/globaldb/global_db_network_audit.go` — Reference for persisted network audit patterns.
- `internal/task/` — Task manager methods and task-store idempotency/event surfaces consumed by network ingress.

### Dependent Files
- `internal/api/core/network.go` — May later expose task-aware network views and filters.
- `internal/observe/` — Will consume network-originated task metrics and audit signals.

### Related ADRs
- [ADR-004: Support Optional Task-to-Network-Channel Binding](../adrs/adr-004.md) — Governs channel binding, mismatch rejection, and stale-channel handling.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs identity/origin derivation for network-originated writes.

## Deliverables
- Network ingress integration for task create/update/run operations.
- Channel-binding validation, mismatch handling, and stale-channel policy enforcement.
- Audit and idempotency coverage for network-originated task writes.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for network peer task flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify ingress with a mismatched channel is rejected for a channel-bound task.
  - [ ] Verify stale channel snapshots are handled according to the revised spec without mutating immutable task history unexpectedly.
  - [ ] Verify duplicate network writes with the same idempotency key resolve to a single canonical task-domain effect.
- Integration tests:
  - [ ] Verify a network peer can create a task with `network_channel` binding and later enqueue a run through the validated manager path.
  - [ ] Verify mismatched network-channel ingress records an audit event and does not mutate the target task.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Network peers can participate in the task domain safely as first-class writers
- Channel-aware task flows behave predictably across peer ingress, retries, and audit inspection

---
status: pending
title: Add typed delivery targets and outbound target resolution seam
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 03: Add typed delivery targets and outbound target resolution seam

## Overview

Add the typed `DeliveryTarget` model and the core-side resolver that turns channel instance defaults plus explicit destination fields into one canonical outbound target. This task intentionally stops at the channel seam so future automation integration has a stable target contract without requiring `internal/automation` to exist first.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST introduce a typed `DeliveryTarget` model and resolver in `internal/channels/` that supports `channel_instance_id`, `peer_id`, `thread_id`, `group_id`, and `mode` exactly as defined in the TechSpec.
2. MUST resolve outbound targets using channel-instance metadata and explicit overrides instead of free-form platform strings.
3. MUST validate target completeness and incompatible field combinations before any transport or automation layer attempts delivery.
4. SHOULD remain independent of a concrete automation runtime so future automation code can consume this seam without forcing undeclared cross-feature dependencies now.
</requirements>

## Subtasks
- [ ] 3.1 Add the typed `DeliveryTarget` model and validation helpers to `internal/channels/`
- [ ] 3.2 Implement target resolution from channel instance defaults plus explicit outbound fields
- [ ] 3.3 Add mode validation and normalization for direct-send, reply, and future target variants used by channels
- [ ] 3.4 Add unit and integration tests for target resolution and validation

## Implementation Details

Follow the TechSpec sections "DeliveryTarget", "Automation", and "Technical Considerations". This task should produce a reusable channel-side resolver seam only; do not introduce direct dependencies on the unimplemented automation runtime in `internal/automation`.

### Relevant Files
- `internal/api/contract/contract.go` — Shared transport DTOs added later should map to this resolver output rather than inventing a parallel target shape
- `internal/extension/contract/host_api.go` — The negotiated channel-delivery contract will later embed resolved target data
- `internal/cli/client.go` — Future CLI operations such as test delivery and route inspection will consume the canonical target model
- `.compozy/tasks/automation/_tasks.md` — Existing automation planning shows this resolver is a future consumer seam, not a same-task dependency

### Dependent Files
- `internal/daemon/boot.go` — Daemon composition later injects this resolver into channel runtime services
- `internal/api/httpapi/routes.go` — Transport handlers later accept or return delivery targets through the API surface
- `internal/session/manager_prompt.go` — Delivery projection later needs one canonical target object for outbound sends

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Keeps outbound governance in the daemon substrate
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Requires delivery targets to resolve through daemon-owned instance identity

## Deliverables
- Typed `DeliveryTarget` model and outbound target resolver in `internal/channels/`
- Validation logic for target completeness and incompatible field combinations
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for outbound target resolution without an automation runtime dependency **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Resolving a target with only `channel_instance_id` and explicit `peer_id` produces a valid direct target
  - [ ] Validation rejects a target that omits a required destination field for the chosen mode
  - [ ] Validation rejects incompatible combinations such as a thread-only target without a peer or group anchor when the mode requires one
  - [ ] Target resolution preserves explicit overrides instead of silently replacing them with instance defaults
- Integration tests:
  - [ ] A channel instance with delivery defaults resolves one canonical outbound target without importing any automation package
  - [ ] Workspace-scoped target resolution does not leak defaults or identifiers from a different channel instance or scope
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Outbound channel delivery can reference one canonical target object instead of free-form strings
- Future automation work has a stable channel-side resolver seam to integrate with

---
status: completed
title: "Add cross-provider multi-instance recovery and conformance coverage"
type: backend
complexity: critical
dependencies:
  - task_09
  - task_10
  - task_11
  - task_12
  - task_13
  - task_14
  - task_15
  - task_16
---

# Task 17: Add cross-provider multi-instance recovery and conformance coverage

## Overview

Close the feature with system-level proof that the provider-scoped substrate behaves consistently across the delivered providers. This task adds the cross-provider integration and conformance matrix for multi-instance ownership, restart recovery, auth degradation, DM policy, and classified retry behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add cross-provider integration coverage for provider-scoped runtime behavior once the real providers are implemented.
2. MUST verify that multiple bridge instances owned by one provider process can ingest, deliver, restart, and report state without violating route ownership or authorization rules.
3. MUST cover DM policy enforcement, structured degradation reporting, and classified retry or recovery behavior across representative providers.
4. SHOULD leave the repository with one reusable conformance matrix that future providers can extend instead of reinventing end-to-end coverage.
</requirements>

## Subtasks

- [x] 17.1 Add multi-instance integration scenarios that exercise shared provider runtimes across representative providers
- [x] 17.2 Add restart and recovery scenarios covering owned-instance cache rehydration and delivery continuity
- [x] 17.3 Add DM policy, auth degradation, and classified retry scenarios across representative providers
- [x] 17.4 Consolidate shared conformance reporting and provider coverage documentation

## Implementation Details

Follow the TechSpec sections "Testing Approach", "Integration Tests", and "Verification Targets". This task should focus on system proof and conformance evidence, not on adding new provider capabilities beyond what earlier tasks already introduced.

### Relevant Files

- `internal/extensiontest/bridge_adapter_harness.go` — Shared conformance harness should become the backbone of the cross-provider matrix
- `internal/daemon/daemon_integration_test.go` — Daemon-level integration coverage can exercise provider-scoped runtime recovery and routing ownership
- `internal/extension/bridge_delivery_integration_test.go` — Delivery-path integration tests should cover provider-scoped runtimes and representative providers
- `internal/api/httpapi/bridges_integration_test.go` — Bridge-management integration coverage can validate state, health, and degradation surfaces end to end

### Dependent Files

- `docs/plans/2026-04-15-bridge-adapters-design.md` — Final verification evidence may require follow-up design notes or implementation references
- `.compozy/tasks/bridge-adapters/_techspec.md` — The task set should leave the verification targets in the spec materially satisfied
- future `extensions/bridges/*` providers — New providers should be able to plug into this conformance matrix later

### Reference Sources (.resources/)

- `.resources/hermes/tests/gateway/` — Hermes 180+ gateway test files covering per-platform tests, shutdown, reconnect, session isolation, delivery, and mock patterns; reference for cross-provider test organization
- `.resources/hermes/gateway/platforms/ADDING_A_PLATFORM.md` — Hermes 16-item integration checklist; useful as a conformance matrix template
- `.resources/openclaw/src/channels/plugins/contracts/` — OpenClaw contract testing: inbound payload contract, outbound payload contract, config contract, actions contract; reference for plugin-level conformance verification
- `.resources/goclaw/tests/integration/abort_router_concurrent_test.go` — GoClaw race condition verification test; reference for concurrent access testing patterns
- `.resources/openclaw/src/gateway/channel-health-monitor.ts` — OpenClaw channel health monitoring with status degradation tracking and audit trails; reference for cross-provider health conformance

### Related ADRs

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Cross-provider tests must verify provider-scoped multiplexing and recovery
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Cross-provider tests must verify ingress hardening, retry classification, and recovery behavior
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Cross-provider tests should verify the approved bridge v1 scope without creeping into deferred capabilities

## Deliverables

- Cross-provider integration suites for multi-instance runtime behavior, recovery, DM policy, and degraded-state reporting
- Consolidated conformance scenarios reusable by current and future providers
- Documentation or test metadata showing which providers satisfy which verification targets
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for cross-provider recovery and conformance behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] shared conformance reporting aggregates provider results consistently across multiple runtime instances
  - [x] classified retry and degradation expectations are asserted consistently across representative providers
- Integration tests:
  - [x] one provider process owning multiple bridge instances can ingest and deliver for each instance without cross-instance leakage
  - [x] provider restart rehydrates owned-instance state and resumes delivery or ingest correctly for representative providers
  - [x] DM policy enforcement rejects unauthorized direct-message ingress and allows authorized ingress according to the configured policy
  - [x] auth failures and rate limits surface structured degradation reasons and the expected state transitions across representative providers
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The bridge-adapters feature has cross-provider proof for multi-instance runtime behavior and recovery
- Future provider work has a reusable conformance matrix instead of one-off end-to-end tests

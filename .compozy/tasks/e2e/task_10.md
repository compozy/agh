---
status: completed
title: Browser network operator flow
type: test
complexity: high
dependencies:
  - task_08
  - task_03
---

# Task 10: Browser network operator flow

## Overview

Add the browser E2E scenario that proves an operator can create and inspect a network collaboration flow through the shipped Network UI. This task treats the browser as the operator proof and reuses the runtime network lane from `task_03` as the backend source of truth.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a Playwright scenario that covers opening the Network page, creating a channel through the UI, inspecting peers, and observing channel message history and timeline state.
2. MUST rely on the real runtime network behavior established by `task_03` rather than faking protocol truth inside the browser.
3. MUST assert browser-visible outcomes such as channel creation visibility, peer materialization, message/timeline rendering, and reload or navigation continuity.
4. MUST keep protocol-level correlation assertions out of this task; those belong to the runtime network lane.
5. SHOULD add only minimal selector stabilization to the Network route if the existing UI surface is not reliable enough for browser automation.
</requirements>

## Subtasks
- [x] 10.1 Seed the browser/runtime harness with network-enabled daemon state suitable for an operator-visible collaboration flow.
- [x] 10.2 Implement the browser scenario for channel creation and channel inspection.
- [x] 10.3 Add peer-visibility and message/timeline assertions for the resulting network collaboration state.
- [x] 10.4 Add reload or cross-navigation assertions that prove the UI can recover and re-read the same network state.
- [x] 10.5 Add selector-stability adjustments only where the shipped route surface requires them.

## Implementation Details

See TechSpec sections "PR-Required Browser E2E", "Runtime Data Flow", and "Technical Considerations". This task is intentionally user-visible; it should not restate low-level RFC assertions already owned by `task_03`.

### Relevant Files
- `web/src/routes/_app/network.tsx` — primary browser route for channel creation and peer inspection.
- `web/src/systems/network/components/network-create-channel-dialog.tsx` — channel-creation surface used by the operator flow.
- `web/src/systems/network/hooks/use-network.ts` — runtime-backed network query behavior visible in the UI.
- `web/src/systems/network/hooks/use-network-actions.ts` — create-channel mutation path used by the route.
- `web/src/systems/network/adapters/network-api.ts` — browser transport surface for network operations.
- `web/e2e/fixtures/runtime.ts` — shared browser/runtime fixture that seeds the network-enabled daemon.

### Dependent Files
- `web/e2e/network.spec.ts` — new browser network operator scenario.
- `web/e2e/fixtures/selectors.ts` — optional shared selectors if the route surface needs stable handles.
- `web/src/routes/_app/network.tsx` — may need minimal test-surface stabilization if existing selectors are insufficient.
- `internal/daemon/daemon_integration_test.go` — runtime network truth source used to seed or cross-check browser-visible outcomes.
- `Makefile` — later browser lane wiring must include this scenario in the browser E2E target set.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — This task is a browser-lane operator proof over an existing runtime-truth surface.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Browser assertions stay UI-visible while the runtime lane owns protocol truth.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Network inspection is an in-scope shipped browser surface.

## Deliverables
- Browser E2E scenario for channel creation, peer inspection, and network timeline visibility
- Seeded browser/runtime fixture support for network-enabled operator flows
- Minimal selector or route-surface stabilization needed for reliable browser assertions
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for the browser network operator workflow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Network browser fixture seeding builds deterministic channel and peer state for the operator scenario
  - [x] Selector helpers locate the create-channel dialog, channel list, peer list, and detail panels consistently
  - [x] Browser artifact capture records enough route context to explain a failed network operator flow
- Integration tests:
  - [x] Operator can open the Network page, create a channel through the UI, and see the new channel materialize
  - [x] Operator can inspect peers and observe channel messages or timeline entries sourced from the real runtime
  - [x] Operator can reload or navigate away and back without losing the visible network state for the same scenario
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Browser E2E proves an operator can manage and inspect network collaboration through the shipped UI
- Runtime network truth remains owned by `task_03`, with browser checks focused on visible behavior
- The network browser scenario runs on the shared Playwright harness with no per-route execution duplication

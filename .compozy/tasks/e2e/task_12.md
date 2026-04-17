---
status: completed
title: Browser bridges operator flow
type: test
complexity: high
dependencies:
  - task_08
  - task_05
---

# Task 12: Browser bridges operator flow

## Overview

Add the browser E2E scenario that proves an operator can manage bridges through the shipped Bridges UI and observe real bridge health and delivery behavior. This task depends on the runtime bridge and extension truth from `task_05`, but keeps its own assertions focused on browser-visible operator outcomes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a Playwright scenario that covers bridge creation or editing, required secret-binding or configuration management, health-stream visibility, and test delivery or equivalent operator action exposed in the UI.
2. MUST assert browser-visible outcomes such as bridge state, health stream updates, and visible downstream effects after test delivery.
3. MUST rely on the real bridge and extension runtime behavior established by `task_05` instead of simulating delivery or provider-side effects inside the browser.
4. MUST keep browser assertions focused on the shipped Bridges route and related operator surfaces.
5. SHOULD add only minimal selector or state-stabilization hooks to the Bridges route if existing UI affordances are not reliable enough for browser automation.
</requirements>

## Subtasks
- [x] 12.1 Seed the browser/runtime harness with bridge-enabled runtime state and provider prerequisites.
- [x] 12.2 Implement the browser scenario for bridge create/edit and required configuration or secret-binding actions.
- [x] 12.3 Add real health-stream and test-delivery assertions through the shipped UI.
- [x] 12.4 Verify visible downstream bridge state changes after the operator action completes.
- [x] 12.5 Add minimal route-surface stabilization only where the existing Bridges UI needs it.

## Implementation Details

See TechSpec sections "PR-Required Browser E2E", "Integration Points", and "Combined-Flow Follow-Up". The browser lane should observe bridge behavior through the operator surface, while the runtime lane remains the source of truth for ingress and subprocess semantics.

### Relevant Files
- `web/src/routes/_app/bridges.tsx` — primary browser route for bridge management.
- `web/src/systems/bridges/hooks/use-bridge-health-stream.ts` — health-stream browser behavior that must work in a real E2E run.
- `web/src/systems/bridges/components/bridge-create-dialog.tsx` — operator surface for bridge creation and provider selection.
- `web/src/systems/bridges/adapters/bridges-api.ts` — browser transport surface for bridge CRUD, secret bindings, and test delivery.
- `internal/api/httpapi/static.go` — browser hosting path used by the Playwright harness while testing bridge flows.
- `web/e2e/fixtures/runtime.ts` — shared Playwright fixture that seeds the bridge-enabled runtime.

### Dependent Files
- `web/e2e/bridges.spec.ts` — new browser Bridges operator scenario.
- `web/e2e/fixtures/selectors.ts` — optional shared selector helpers for bridge surfaces and health-state waits.
- `web/src/routes/_app/bridges.tsx` — may need minimal test-surface stabilization if current handles are not sufficient.
- `internal/extensiontest/bridge_adapter_harness.go` — runtime truth layer that later browser cross-checks may reference sparingly.
- `Makefile` — later browser lane wiring must include this scenario in the browser E2E target set.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — This task is a browser-lane operator proof over a runtime-managed bridge system.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Browser-visible health and delivery outcomes are the primary assertions.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Bridge management is an in-scope shipped browser surface.

## Deliverables
- Browser E2E scenario for bridge management, health updates, and test delivery
- Browser/runtime fixture support for bridge-enabled operator flows
- Minimal UI stabilization required for reliable browser assertions on the Bridges route
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for the browser bridge operator workflow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Bridge browser fixture seeding provides deterministic provider, bridge, and health-state prerequisites
  - [x] Selector helpers locate bridge create/edit surfaces, health stream indicators, and test-delivery actions consistently
  - [x] Browser artifact capture records enough bridge-route state to explain failed health or delivery flows
- Integration tests:
  - [x] Operator can create or edit a bridge, satisfy required configuration or secret-binding steps, and see the bridge appear in the UI
  - [x] Operator can observe real health-stream updates through the Bridges route in a Playwright run
  - [x] Operator can run test delivery or equivalent bridge action and observe visible downstream state change
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Browser E2E proves an operator can manage bridges through the shipped UI
- Real health and delivery behavior remains backed by runtime bridge and extension truth
- The browser bridges task remains independent once the shared browser harness and runtime bridge lane exist

---
status: completed
title: Browser onboarding and session lifecycle flow
type: test
complexity: high
dependencies:
  - task_08
---

# Task 09: Browser onboarding and session lifecycle flow

## Overview

Add the browser E2E scenario that proves an operator can move from workspace onboarding to a working session lifecycle entirely through the shipped web UI. This task covers the most complete single-user workflow in the current product: onboarding, session creation, prompt streaming, approval, stop/resume, and reload hydration.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a Playwright scenario that covers workspace onboarding or selection, session creation, prompt submission, streaming visibility, stop/resume actions, and page reload hydration.
2. MUST cover the shipped approval UI path when a permission request is surfaced by the running session.
3. MUST assert browser-visible outcomes first, using a second daemon read path only when it materially explains a visible failure.
4. MUST consume the shared browser harness from `task_08` and avoid route-specific execution setup duplication.
5. SHOULD keep onboarding and session lifecycle in one task because they form one real operator journey through the shipped shell.
</requirements>

## Subtasks
- [x] 9.1 Add seeded browser/runtime fixtures for workspace onboarding and session creation.
- [x] 9.2 Implement the end-to-end session lifecycle scenario in Playwright.
- [x] 9.3 Cover approval UI behavior within the same operator journey.
- [x] 9.4 Add reload and hydration assertions for transcript continuity and session state.
- [x] 9.5 Add focused selector-stability or shell-surface adjustments only where the real route needs them.

## Implementation Details

See TechSpec sections "PR-Required Browser E2E", "Browser Data Flow", and "Technical Considerations". This task should remain a user-visible workflow proof, not a browser replay of daemon-truth protocol assertions.

### Relevant Files
- `web/src/routes/_app.tsx` — workspace onboarding and application shell boundary for the operator flow.
- `web/src/routes/_app/session.$id.tsx` — session chat route that must support streaming, approval, stop/resume, and hydration checks.
- `web/src/systems/session/components/permission-prompt.tsx` — approval UI surface used by the browser flow.
- `web/src/systems/session/hooks/use-session-chat.ts` — chat streaming and permission event handling path visible in the UI.
- `web/src/systems/session/adapters/session-api.ts` — browser-side session transport surface used by the route.
- `web/e2e/fixtures/runtime.ts` — shared Playwright fixture consumed by the new workflow scenario.

### Dependent Files
- `web/e2e/session-onboarding.spec.ts` — new browser E2E scenario for the full onboarding/session workflow.
- `web/e2e/fixtures/selectors.ts` — optional shared selectors if route-level stability helpers are needed.
- `web/src/routes/_app.tsx` — may need stable test hooks only if the existing shell surface is insufficient.
- `web/src/routes/_app/session.$id.tsx` — may need stable test hooks only if the existing route surface is insufficient.
- `Makefile` — later lane wiring must include this scenario in the browser E2E target set.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — This task implements one of the browser lane’s operator-journey proofs.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Browser assertions should remain UI-visible and use backend reads only for explanation.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Session chat and approval are in-scope shipped browser surfaces.

## Deliverables
- Browser E2E scenario for onboarding, session creation, prompt streaming, approval, stop/resume, and reload hydration
- Stable browser fixture data for seeded workspace and session-state setup
- Minimal selector or test-surface stabilization needed for reliable operator-journey assertions
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for the full browser session lifecycle flow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Session browser fixture seeding creates the required workspace and session state without hidden route assumptions
  - [x] Selector helpers locate onboarding, session, and approval surfaces consistently across the shipped shell
  - [x] Browser artifact capture records the session route state for streaming and hydration failures
- Integration tests:
  - [x] Operator can complete onboarding or workspace selection, create a session, send a prompt, and observe streaming output
  - [x] Operator can resolve a permission prompt through the approval UI and see the session continue
  - [x] Operator can stop and resume the session, reload the page, and still see transcript and session-state continuity
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Browser E2E proves a complete session operator journey through the shipped UI
- Approval and hydration behavior are covered without turning Playwright into a protocol-truth layer
- The browser session flow runs on the shared daemon-served Playwright harness

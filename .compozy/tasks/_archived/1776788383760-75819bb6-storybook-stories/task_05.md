---
status: completed
title: MSW mocks for all 9 systems + handler composition
type: frontend
complexity: high
dependencies:
  - task_01
---

# Task 5: MSW mocks for all 9 systems + handler composition

## Overview
Create per-system `mocks/{handlers.ts,fixtures.ts,index.ts}` triples for all nine systems (agent, automation, bridges, daemon, knowledge, network, session, skill, workspace) and wire the default handler set into `web/.storybook/preview.ts`. This gates every system-story task and establishes the reusable fixture surface that Vitest and future Playwright runs will share.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `web/src/systems/<name>/mocks/handlers.ts`, `fixtures.ts`, and `index.ts` for every system listed above, even when the system currently has no outbound HTTP (export `handlers: HttpHandler[] = []` for symmetry).
- MUST type every fixture against the existing adapter response type (imported from `web/src/systems/<name>/adapters` or `web/src/systems/<name>/types`). Fixtures MUST NOT import from hooks, stores, or components.
- MUST expose a barrel `index.ts` re-exporting `handlers` and named fixtures; system public barrels (`web/src/systems/<name>/index.ts`) MUST NOT re-export from `mocks/`.
- MUST compose a default handler array inside `web/.storybook/preview.ts` that spreads every system's `handlers` into `parameters.msw.handlers`.
- MUST use realistic but minimal fixtures (2–3 list items per collection, 1 representative detail object).
- MUST handle the AGH daemon's `/api/**` routes under the exact paths consumed by each system's adapter; path drift breaks stories silently.
- MUST NOT introduce new runtime dependencies; only `msw` (already added in task_01) is allowed.
</requirements>

## Subtasks
- [x] 5.1 Inventory every `/api/**` path each system calls from its `adapters/` and codify handlers that return representative payloads.
- [x] 5.2 Write typed fixture constants per system, sourcing types from the existing adapter/type modules.
- [x] 5.3 Export `handlers` plus fixtures via a `mocks/index.ts` barrel for each of the nine systems.
- [x] 5.4 Update `web/.storybook/preview.ts` to import all nine barrels and compose `parameters.msw.handlers` with the union.
- [x] 5.5 Smoke-test the web Storybook by loading an existing `design-system` story with DevTools open and confirming no unhandled-request warnings.

## Implementation Details
See TechSpec "Per-system MSW contract" in the Core Interfaces section for the shape of each `handlers.ts`. Paths must match the adapter code; prefer importing path constants when they exist. For systems without external HTTP (daemon's `connection-status` is view-model over a query but may poll `/api/daemon/health`), inspect the adapter and include at least one handler that keeps the query resolved to success state.

### Relevant Files
- `web/src/systems/<name>/adapters/*-api.ts` — source of API path strings and response types.
- `web/src/systems/<name>/types.ts` — source of typed fixture shapes.
- `web/.storybook/preview.ts` — target of the handler composition.
- `.compozy/tasks/storybook-stories/_techspec.md` — authoritative per-system handler contract.

### Dependent Files
- `task_06`..`task_10` — every system-story task depends on this task's handlers and fixtures.
- `web/src/systems/<name>/index.ts` — verified to confirm no mocks are re-exported by default.

### Related ADRs
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — Mandates MSW as the data layer.
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md) — Places mocks inside each system folder.

## Deliverables
- 27 new files: `handlers.ts`, `fixtures.ts`, `index.ts` per system × 9 systems.
- Updated `web/.storybook/preview.ts` composing the default handler set.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for MSW handler coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Each system's `handlers.ts` exports an array typed as `HttpHandler[]` and compiles under `tsgo --noEmit`.
  - [ ] Every fixture constant assigns to its corresponding adapter type without `as any` or casts.
  - [ ] The composed default handler set in `preview.ts` exposes at least one handler per system whose adapter makes an HTTP call.
- Integration tests:
  - [ ] Loading a web Storybook story with DevTools records zero "unhandled request" warnings from MSW for every path any system adapter calls.
  - [ ] A dedicated sanity test imports every system's `handlers` barrel and asserts no duplicate URL+method pairs (catches accidental overrides).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Nine `mocks/` folders present and barrel-exposed.
- `preview.ts` default handler set covers every adapter path.
- `web build-storybook` exits 0; running the existing `design-system` stories emits no MSW warnings.

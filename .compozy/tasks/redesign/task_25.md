---
status: pending
title: Rewrite Bridges domain (list + detail)
type: frontend
complexity: medium
dependencies:
  - task_13
  - task_14
---

# Task 25: Rewrite Bridges domain (list + detail)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/bridges/**` as a pure visual refresh over `@agh/ui` primitives. The `/bridges` route is a split-pane view: left list of integrations (Slack, Email, Linear, GitHub, …) grouped by provider with search, right detail showing provider info, connection status, four headline `Metric` tiles, and a recent event stream `Table`. All TanStack Query hooks, adapters, stores, and MSW fixtures stay untouched — only visual chrome changes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite every component under `web/src/systems/bridges/components/**` and the route `web/src/routes/_app/bridges.tsx` using only `@agh/ui` primitives plus domain code.
- MUST compose the page as `PageHeader` (title + count + scope `Pills` + primary action) above a `SplitPane` whose `list` holds the provider-grouped list and `detail` holds the selected bridge.
- MUST render bridge detail as `Section` blocks: provider summary (name, scope, status via `StatusDot`), four-tile `Metric` row (events/24h, success rate, last delivery, active routes), recent event stream `Table`.
- MUST use `SearchInput` for list filtering and `Empty` for every no-bridges / no-results / disabled / errored state — including the zero-bridges provider-cards empty state.
- MUST keep `BridgeCreateDialog`, `BridgeEditDialog`, and `BridgeTestDeliveryDialog` on the `@agh/ui` `Dialog` primitive composed from `Field` primitives — no raw `<form>` or `@/components/ui/*` wrappers.
- MUST preserve existing behavior from `useBridgesPage` and every `useBridge*` hook — props shape may shift but data sources stay identical.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*` — both folders are deleted after Phase 2.
- SHOULD replace `bridge-provider-card.tsx` grid markup with a composition over `@agh/ui` primitives used consistently with other domain empty states.
</requirements>

## Subtasks

- [ ] 25.1 Audit current components + `useBridgesPage` view-model and list every prop, state branch, and test id to preserve.
- [ ] 25.2 Rewrite `bridges.tsx` around `PageHeader` + scope `Pills` + `SplitPane`, removing `WorkspacePageShell` and `PillButton`.
- [ ] 25.3 Rewrite `bridge-list-panel.tsx` as `SearchInput` + grouped list rows using `StatusDot` + provider `KindChip`, with `Empty` states for no-results and errors.
- [ ] 25.4 Rewrite `bridge-detail-panel.tsx` as `Section` blocks with a four-tile `Metric` row and event stream `Table`.
- [ ] 25.5 Rewrite `bridge-create-dialog.tsx`, `bridge-edit-dialog.tsx`, and `bridge-test-delivery-dialog.tsx` on `@agh/ui` `Dialog` + `Field`.
- [ ] 25.6 Rewrite `bridge-empty-state.tsx` + `bridge-provider-card.tsx` on `Empty` and shared primitives.
- [ ] 25.7 Update or rewrite `web/src/systems/bridges/components/stories/**` and generate Playwright visual baselines covering default / empty / error / each dialog state.

## Implementation Details

See TechSpec §"Impact Analysis" — `web/src/systems/bridges/**` is a Phase 5 visual rewrite. DESIGN.md §4/§5 govern the visual spec (Pills, StatusDot, Metric, Section). Event stream `Table` and `Metric` row are shared patterns with Network + Automation detail panels.

### Relevant Files

- `web/src/routes/_app/bridges.tsx` — rewrite target; drop `WorkspacePageShell` + `PillButton` imports.
- `web/src/systems/bridges/components/bridge-list-panel.tsx` — rewrite on `SearchInput` + grouped list rows.
- `web/src/systems/bridges/components/bridge-detail-panel.tsx` — rewrite on `Section` + `Metric` + `Table`; drops `Pill` + `Input` locally-imported shapes.
- `web/src/systems/bridges/components/bridge-create-dialog.tsx` — rewrite on `@agh/ui` `Dialog` + `Field`.
- `web/src/systems/bridges/components/bridge-edit-dialog.tsx` — rewrite on `@agh/ui` `Dialog` + `Field`.
- `web/src/systems/bridges/components/bridge-test-delivery-dialog.tsx` — rewrite on `@agh/ui` `Dialog` + `Field`.
- `web/src/systems/bridges/components/bridge-empty-state.tsx` — rewrite on `Empty`.
- `web/src/systems/bridges/components/bridge-provider-card.tsx` — rewrite or absorb into the empty-state composition.

### Dependent Files

- `web/src/hooks/routes/use-bridges-page.ts` — view-model stays; only the shape of props handed to panels may change.
- `web/src/systems/bridges/index.ts` — public barrel; update exports if panel modules split.
- `web/src/systems/bridges/components/stories/**` — stories rewritten against new primitives.
- `web/e2e/**` Playwright suites referencing `data-testid="bridge-*"` — test ids MUST survive.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `web/src/systems/bridges/**` components consuming only `@agh/ui` + domain code.
- Rewritten `web/src/routes/_app/bridges.tsx` wired to `PageHeader` + `SplitPane`.
- Three bridge dialogs (create, edit, test delivery) rebuilt on `@agh/ui` `Dialog` + `Field`.
- Updated Storybook stories for every component under `components/stories/**`.
- Playwright visual snapshot baselines for `/bridges` covering: list default, list empty (no bridges), list filtered-empty, detail default, detail disabled, create dialog, edit dialog, test-delivery dialog **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for scope switch, row select, and dialog submit flow **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `BridgeListPanel` with a non-empty `bridges` list renders one row per bridge and groups rows under their provider heading.
  - [ ] Typing in `SearchInput` filters visible rows and renders the `Empty` no-results state when nothing matches.
  - [ ] Selecting a bridge row calls `onSelectBridge` with the bridge id and marks the row active.
  - [ ] `BridgeDetailPanel` with a bridge renders a `Metric` row containing exactly four tiles (events/24h, success rate, last delivery, active routes).
  - [ ] `BridgeDetailPanel` with `status="disabled"` renders `StatusDot` tone `danger` and disables the "Send Test" button.
  - [ ] `BridgeEmptyState` with `totalBridgeCount=0` renders the `Empty` primitive with the create action bound to `onCreate`.
  - [ ] `BridgeCreateDialog` with invalid form state renders the submit button disabled and calls `onSubmit` only when every required `Field` is valid.
- Integration tests:
  - [ ] Storybook `play()` opens the create dialog, fills provider + route fields, and asserts `onSubmit` receives the expected payload.
  - [ ] Storybook `play()` clicks "Send Test" on a healthy bridge and asserts the test-delivery dialog opens with the bridge id prefilled.
  - [ ] Storybook `play()` with the bridges-error fixture asserts the list panel renders the `Empty` error state.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Zero imports from `@/components/ui/*` or `@/components/design-system/*` anywhere under `web/src/systems/bridges/**` or `web/src/routes/_app/bridges.tsx`.
- Playwright baseline snapshots committed for the eight listed states.
- Every `data-testid="bridge-*"` referenced by existing Playwright e2e specs still resolves.
- `make verify` and `make web-lint` pass.

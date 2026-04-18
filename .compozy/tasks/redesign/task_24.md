---
status: pending
title: Rewrite Automation domain (jobs + triggers)
type: frontend
complexity: high
dependencies:
  - task_13
  - task_14
---

# Task 24: Rewrite Automation domain (jobs + triggers)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/automation/**` as a pure visual refresh over `@agh/ui` primitives. The `/automation` route exposes two tabs (Jobs / Triggers); each is a split-pane view with a searchable list on the left and a detail panel on the right. Jobs detail surfaces cron configuration, success-rate `Metric`, and run history; triggers detail surfaces hook configuration and run history. All TanStack Query hooks, adapters, stores, and MSW fixtures stay untouched — only visual chrome changes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite every component under `web/src/systems/automation/components/**` and the route `web/src/routes/_app/automation.tsx` using only `@agh/ui` primitives plus domain code.
- MUST compose the page as `PageHeader` (title + count + kind `Pills` + scope `Pills` + primary action) above a `SplitPane` whose slots switch between jobs and triggers content.
- MUST render job detail as `Section` blocks for schedule (cron config), stats (`Metric` row: runs, success %, last run, next run), and run history (`Table`).
- MUST render trigger detail as `Section` blocks for hook configuration (source + event + filters) and run history (`Table`).
- MUST use `SearchInput` for list filtering, `StatusDot` for every enabled/disabled/errored signal, and `Empty` for every no-results / loading-fail state.
- MUST move the job + trigger editor into an `@agh/ui` `Dialog` composed from `Field` primitives — no raw `<form>` or `@/components/ui/*` form wrappers.
- MUST preserve existing behavior from `useAutomationPage` and the `useAutomationJob*` / `useAutomationTrigger*` hooks — props shape may shift but data sources stay identical.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*` — both folders are deleted after Phase 2.
- SHOULD keep `automation-job-form.tsx` and `automation-trigger-form.tsx` as focused children of the editor dialog, not as wrappers that own dialog state.
</requirements>

## Subtasks

- [ ] 24.1 Audit current components + `useAutomationPage` view-model and list every prop, state branch, and test id to preserve.
- [ ] 24.2 Rewrite `automation.tsx` around `PageHeader` + kind/scope `Pills` + `SplitPane`, removing `PillButton` and all ad hoc header markup.
- [ ] 24.3 Rewrite `automation-list-panel.tsx` as `SearchInput` + list rows using `StatusDot` + `MonoBadge`, with `Empty` states for no-results and errors.
- [ ] 24.4 Rewrite `automation-detail-panel.tsx` as conditional job/trigger detail with `Section` + `Metric` + `Table` + `StatusDot` + `Pills` for run status filters; replace the local `Empty*` imports from `@/components/ui/empty` with the `@agh/ui` `Empty` primitive.
- [ ] 24.5 Rewrite `automation-editor-dialog.tsx` on the `@agh/ui` `Dialog`, composing `automation-job-form.tsx` and `automation-trigger-form.tsx` on `Field` primitives.
- [ ] 24.6 Rewrite `automation-run-history.tsx` on `Table` + `StatusDot` + `KindChip` (trigger source) and drop any `Pill` usage from `design-system`.
- [ ] 24.7 Update or rewrite `web/src/systems/automation/components/stories/**` and generate Playwright visual baselines covering default / loading / error / empty states for both tabs plus the editor dialog (create + edit).

## Implementation Details

See TechSpec §"Impact Analysis" — `web/src/systems/automation/**` is a Phase 5 visual rewrite. DESIGN.md §4/§5 govern the visual spec (Pills, StatusDot, Metric, Section). Editor dialog uses `@agh/ui` `Dialog` + `Field` per Phase 1 primitive set.

### Relevant Files

- `web/src/routes/_app/automation.tsx` — rewrite target; drop ad hoc header + `PillButton` imports.
- `web/src/systems/automation/components/automation-list-panel.tsx` — rewrite on `SearchInput` + list rows.
- `web/src/systems/automation/components/automation-detail-panel.tsx` — rewrite on `Section` + `Metric` + `Table`; currently imports `@/components/ui/empty` which must go.
- `web/src/systems/automation/components/automation-editor-dialog.tsx` — rewrite on `@agh/ui` `Dialog`.
- `web/src/systems/automation/components/automation-job-form.tsx` — rewrite on `Field` primitives.
- `web/src/systems/automation/components/automation-trigger-form.tsx` — rewrite on `Field` primitives.
- `web/src/systems/automation/components/automation-run-history.tsx` — rewrite on `Table` + `StatusDot` + `KindChip`.

### Dependent Files

- `web/src/hooks/routes/use-automation-page.ts` — view-model stays; only the shape of props handed to panels may change.
- `web/src/systems/automation/components/automation-form-primitives.tsx` — replace or delete once `@agh/ui` `Field` covers the same cases.
- `web/src/systems/automation/index.ts` — public barrel; update exports if panel modules split.
- `web/src/systems/automation/components/stories/**` — stories rewritten against new primitives.
- `web/e2e/**` Playwright suites referencing `data-testid="automation-*"` — test ids MUST survive.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `web/src/systems/automation/**` components consuming only `@agh/ui` + domain code.
- Rewritten `web/src/routes/_app/automation.tsx` wired to `PageHeader` + `SplitPane`.
- Editor dialog rebuilt on `@agh/ui` `Dialog` + `Field` for both jobs and triggers.
- Updated Storybook stories for every component under `components/stories/**`.
- Playwright visual snapshot baselines for `/automation` covering: jobs default, jobs empty, triggers default, triggers empty, automation error, editor-create dialog, editor-edit dialog **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for tab switch, scope switch, and editor-dialog submit **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Toggling the kind `Pills` from jobs to triggers invokes `handleTabChange("triggers")` and swaps the list rows.
  - [ ] Toggling the scope `Pills` to `workspace` invokes `handleScopeChange("workspace")` and filters the list to workspace-scoped rows.
  - [ ] `AutomationListPanel` with `isLoading=true` renders the skeleton rows rather than the `Empty` state.
  - [ ] `AutomationDetailPanel` with a job renders a `Metric` row whose "success rate" tile reflects the computed percentage from the fixture.
  - [ ] `AutomationDetailPanel` with a trigger renders the hook `Section` with the trigger source as a `KindChip` and the event name in a `MonoBadge`.
  - [ ] `AutomationRunHistory` with zero runs renders the `Empty` state and never mounts the `Table`.
  - [ ] `AutomationEditorDialog` with invalid form state renders the submit button disabled and calls `onSubmit` only when every required `Field` is valid.
- Integration tests:
  - [ ] Storybook `play()` opens the editor dialog via the primary action button, fills the job form, and asserts `onSubmit` receives the expected payload.
  - [ ] Storybook `play()` selects a trigger row and asserts the detail panel renders the trigger hook `Section` plus run history `Table`.
  - [ ] Storybook `play()` with a fetch-error fixture asserts the list panel renders the `Empty` error state with a retry affordance.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Zero imports from `@/components/ui/*` or `@/components/design-system/*` anywhere under `web/src/systems/automation/**` or `web/src/routes/_app/automation.tsx`.
- Playwright baseline snapshots committed for the seven listed states.
- Every `data-testid="automation-*"` referenced by existing Playwright e2e specs still resolves.
- `make verify` and `make web-lint` pass.

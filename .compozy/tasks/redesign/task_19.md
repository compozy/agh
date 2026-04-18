---
status: pending
title: Rewrite Tasks domain forms and run detail route
type: frontend
complexity: medium
dependencies:
  - task_17
---

# Task 19: Rewrite Tasks domain forms and run detail route

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the remaining Tasks-domain surfaces — the create modal, edit route, and run detail route — against `@agh/ui` form and display primitives. Form validation, mutation hooks, and route loaders are preserved bit-for-bit; only the visual primitives change. This closes out the Tasks domain rewrite (Phase 3) alongside tasks 17 and 18.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `tasks-create-modal.tsx` on `@agh/ui` `Dialog` + `Field` + `Input` + `Textarea` + `Combobox` + `Button`, preserving the existing `use-tasks-create-modal-form.ts` hook signature and validation rules.
- MUST rewrite `task-editor-surface.tsx` (mounted by `web/src/routes/_app/tasks.$id.edit.tsx`) on the same `Field` + `Input` + `Textarea` + `Combobox` primitives plus `PageHeader` for the title bar and `Button` for save/cancel.
- MUST rewrite `task-run-detail-header.tsx`, `task-run-detail-panels.tsx`, and `task-run-detail-session-link.tsx` on `PageHeader`, `MonoBadge`, `StatusDot`, `Section`, `Metric`, `CodeBlock`, and `Table`; the route stays at `/tasks/$id/runs/$runId`.
- MUST NOT change any form validation logic, TanStack Query mutation hook, or route loader — presentational surface only.
- MUST NOT import from `web/src/components/ui/**` or `web/src/components/design-system/**`.
- MUST keep the existing Vitest tests under `web/src/routes/_app/-tasks.$id.edit.test.tsx`, `-tasks.$id.runs.$runId.test.tsx`, `-tasks.new.test.tsx`, and `web/src/systems/tasks/components/tasks-create-modal.test.tsx` green; update a test only when the component's public props or DOM structure genuinely change.
- MUST render the run detail header with: title + `MonoBadge` run id + `StatusDot` + duration metric + right-side "Open session" link to `/session/$sessionId`.
- MUST render the run detail body as `Section` blocks: Overview (`Metric` row), Output (`CodeBlock`), Events (`Table`), Linked session (`task-run-detail-session-link.tsx`).
- SHOULD reuse the detail header and tab patterns from task 17 where the shape matches rather than introducing new compositions.
</requirements>

## Subtasks

- [ ] 19.1 Rewrite `tasks-create-modal.tsx` on `Dialog` + form primitives; verify `use-tasks-create-modal-form.ts` callers see the same prop shape.
- [ ] 19.2 Rewrite `task-editor-surface.tsx` on the same form primitives + `PageHeader` for `/tasks/$id/edit`.
- [ ] 19.3 Rewrite `task-run-detail-header.tsx` + `task-run-detail-panels.tsx` + `task-run-detail-session-link.tsx` on `PageHeader`, `Section`, `Metric`, `CodeBlock`, `Table`.
- [ ] 19.4 Update the three route files (`tasks.new.tsx`, `tasks.$id.edit.tsx`, `tasks.$id.runs.$runId.tsx`) to render the rewritten components; keep loaders and search-param schemas unchanged.
- [ ] 19.5 Rewrite Storybook stories for the modal, editor surface, and run detail covering empty/pending/success/error/validation-error states.
- [ ] 19.6 Run `make verify` and `pnpm test:visual`; commit Playwright baselines for `/tasks/new`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId`.

## Implementation Details

See TechSpec §"Core Interfaces" for the `Dialog`, `Combobox`, and `Table` contracts and §"Impact Analysis" for the Tasks rewrite scope. DESIGN.md §4 defines form visuals (field spacing, label typography, combobox chrome). Validation rules live in `use-tasks-create-modal-form.ts` and the editor hook — do not touch them.

Field set for the create modal and editor (unchanged from today):

- Title (`Input`, required, min 3 chars)
- Description (`Textarea`)
- Workspace (`Combobox` bound to workspace list query)
- Agent (`Combobox` bound to agent list query)
- Priority (`Combobox`: low / normal / high)
- Tags (`Combobox` multi-select)

### Relevant Files

- `web/src/systems/tasks/components/tasks-create-modal.tsx` — rewrite target (modal + form body).
- `web/src/systems/tasks/components/use-tasks-create-modal-form.ts` — consumed as-is; do not modify.
- `web/src/systems/tasks/components/task-editor-surface.tsx` — rewrite target (edit route surface).
- `web/src/systems/tasks/components/task-run-detail-header.tsx` — rewrite target.
- `web/src/systems/tasks/components/task-run-detail-panels.tsx` — rewrite target.
- `web/src/systems/tasks/components/task-run-detail-session-link.tsx` — rewrite target.
- `web/src/routes/_app/tasks.new.tsx` — mounts the create modal.
- `web/src/routes/_app/tasks.$id.edit.tsx` — mounts the editor surface.
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx` — mounts the run detail panels.

### Dependent Files

- `web/src/systems/tasks/hooks/**` mutation hooks — consumed as-is.
- `web/src/integrations/tanstack-query/**` MSW fixtures — consumed as-is.
- Task 20 (Session thread) consumes the run detail → session link flow rewritten here.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 3, step 3 (closes Tasks domain).
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `tasks-create-modal.tsx`, `task-editor-surface.tsx`, and the three `task-run-detail-*.tsx` files.
- Updated route files for `tasks.new.tsx`, `tasks.$id.edit.tsx`, `tasks.$id.runs.$runId.tsx`.
- Storybook stories for each covering empty, populated, pending, error, and validation-error states.
- Playwright visual baselines for the three routes.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests for the create submission flow and the run detail → session navigation **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `tasks-create-modal.tsx` renders `Dialog` with the title "Create task" and six labeled `Field`s (Title, Description, Workspace, Agent, Priority, Tags).
  - [ ] `tasks-create-modal.tsx` disables the Submit `Button` when the Title field is empty and re-enables it when a 3+ character title is typed.
  - [ ] `tasks-create-modal.tsx` renders a validation error under the Title `Field` when the form hook returns `{ title: "Title is required" }` and clears it after a valid keystroke.
  - [ ] `task-editor-surface.tsx` pre-fills every `Field` from the incoming `task` prop and marks the form dirty only after a user keystroke.
  - [ ] `task-run-detail-header.tsx` renders `PageHeader` with title + `MonoBadge` run id + `StatusDot` with tone mapped from `run.status`.
  - [ ] `task-run-detail-panels.tsx` renders the Output `CodeBlock` with language `json` when `run.output.kind === "json"` and with language `text` otherwise.
  - [ ] `task-run-detail-panels.tsx` renders the Events `Table` with one row per entry in `run.events` and an `Empty` primitive when the list is empty.
  - [ ] `task-run-detail-session-link.tsx` renders nothing when `run.sessionId` is undefined.
- Integration tests:
  - [ ] Storybook `play()` in `tasks-create-modal.stories.tsx` fills every required field, clicks Submit, asserts the create mutation hook fires once with the expected payload, and the dialog closes on success.
  - [ ] Vitest route test in `-tasks.new.test.tsx` submits with an empty Title, asserts the Title `Field` renders the validation message, and the mutation hook is NOT called.
  - [ ] Vitest route test in `-tasks.$id.runs.$runId.test.tsx` clicks the Open session affordance, asserts the router navigates to `/session/$sessionId` with the run's linked session id.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80% across the rewritten files.
- `rg "@/components/(ui|design-system)/" web/src/systems/tasks/components/{tasks-create-modal,task-editor-surface,task-run-detail-*}.tsx` returns zero matches.
- Form validation behavior unchanged: every pre-existing form test in `tasks-create-modal.test.tsx` passes without being updated.
- Playwright baselines for `/tasks/new`, `/tasks/$id/edit`, and `/tasks/$id/runs/$runId` committed and match within the 0.1% pixel-diff threshold.
- Tasks domain (tasks 17, 18, 19) is visually complete; no remaining file under `web/src/systems/tasks/components/` imports from the deleted primitive folders.
- `make verify` passes.

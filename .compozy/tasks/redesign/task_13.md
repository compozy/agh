---
status: pending
title: Rewrite app-sidebar on @agh/ui Sidebar
type: refactor
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 13: Rewrite app-sidebar on @agh/ui Sidebar

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/components/app-sidebar.tsx` as a thin composition over `@agh/ui` `Sidebar`. The new shell has four slots: workspace rail (icon column), header (wordmark + `ALPHA` chip), nav (section headers + agent tree + nav rows), footer (connection indicator + version + settings gear). Domain content (workspace list, agent tree) stays inside `web/src/components/app-sidebar.tsx`; the visual chrome comes from `@agh/ui`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `web/src/components/app-sidebar.tsx` as a composition of `@agh/ui` `Sidebar` + `StatusDot` + `ConnectionIndicator` + `Pills` where appropriate.
- MUST pass the workspace switcher (icon rail), app wordmark + `ALPHA` chip, agent tree + nav items, and connection footer via the `Sidebar` slots (rail, header, nav, footer).
- MUST integrate with existing `web/src/stores/sidebar-store.ts` and `web/src/hooks/use-sidebar-store.ts` for collapse state.
- MUST preserve all existing behaviors: active route highlighting, workspace switching, agent tree expand/collapse, session list per agent, "+" new workspace, settings gear link.
- MUST NOT introduce any new `@/components/ui/*` imports (that folder is gone after task 08).
- MUST use Lucide icons with DESIGN.md stroke and size conventions.
- MUST use `motion` only if `Sidebar` itself does not already provide the collapse animation.
- SHOULD delete dead helper code that previously lived in `app-sidebar.tsx` and is no longer needed after consolidation.
</requirements>

## Subtasks

- [ ] 13.1 Audit the existing `app-sidebar.tsx` and the sidebar store to understand what state + behavior must survive the rewrite.
- [ ] 13.2 Compose the new `app-sidebar.tsx` on top of `@agh/ui` `Sidebar` with the four slots.
- [ ] 13.3 Wire active route highlighting via TanStack Router's `useMatchRoute` or equivalent.
- [ ] 13.4 Wire workspace switching + agent tree expand/collapse via the existing stores.
- [ ] 13.5 Write or update Storybook stories for the composed `AppSidebar` covering: default, collapsed, no workspaces, many workspaces, disconnected state.
- [ ] 13.6 Run `make verify` and verify the dev server loads every top-level route.

## Implementation Details

See TechSpec "Impact Analysis" — `web/src/components/app-sidebar.tsx` is a full rewrite. DESIGN.md §4 "Sidebar (Operator UI)" is the visual spec. The mock `docs/design/web-inspiration/src/sidebar.jsx` shows the slot composition.

### Relevant Files

- `web/src/components/app-sidebar.tsx` — rewrite target.
- `web/src/stores/sidebar-store.ts` — collapse + active workspace state.
- `web/src/hooks/use-sidebar-store.ts` — consumer hook.
- `web/src/systems/agent/components/*` — agent tree group consumed as a slot child.
- `web/src/systems/workspace/components/*` — workspace switcher consumed as a rail slot.
- `packages/ui/src/components/sidebar.tsx` — new primitive (task 05).
- `docs/design/web-inspiration/src/sidebar.jsx` — reference composition.

### Dependent Files

- Every route in `web/src/routes/_app/**` renders inside this sidebar.
- Task 14 (root layout) composes this `AppSidebar` at the app shell level.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)

## Deliverables

- Rewritten `app-sidebar.tsx` composed from `@agh/ui` primitives.
- Storybook stories covering default / collapsed / empty / disconnected.
- Playwright snapshot baseline for each story variant.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for collapse + workspace switch **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Renders the wordmark + `ALPHA` chip in the header slot.
  - [ ] Renders the workspace rail with one circle per workspace and the "+" create affordance.
  - [ ] Clicking a workspace icon calls the store's `setActiveWorkspace` action.
  - [ ] Nav section headers render in JetBrains Mono 11px uppercase.
  - [ ] Active route row renders the 3px left accent bar + primary text color.
  - [ ] Footer `ConnectionIndicator` reflects the daemon connection state.
- Integration tests:
  - [ ] Storybook `play()` toggles collapse and asserts width transition + preserved rail.
  - [ ] Storybook `play()` switches workspace and asserts agent tree reflects new workspace content.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `app-sidebar.tsx` imports only `@agh/ui` for primitives and local domain components for content.
- Dev server renders every top-level route with the new sidebar.
- Playwright baseline snapshots committed for the four shell states.
- `make verify` passes.

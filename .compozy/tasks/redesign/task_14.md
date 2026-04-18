---
status: completed
title: Rewrite root layout + route-level motion
type: frontend
complexity: high
dependencies:
  - task_05
  - task_06
---

# Task 14: Rewrite root layout + route-level motion

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/routes/__root.tsx` and `web/src/routes/_app.tsx` to compose the sticky-blur app header, the `Sidebar`-rooted left column, the content column using `SplitPane` where needed, and route-level transitions via `motion`'s `AnimatePresence`. This task also wires `<UIProvider>` at the app root so the motion config becomes global.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST wrap the app in `<UIProvider reducedMotion="user">` at `web/src/main.tsx` (or `__root.tsx`, whichever hosts the entry).
- MUST rewrite `__root.tsx` to render the global app shell: sticky header (bg `rgba(20,19,18,0.92)` + `backdrop-blur-xl`), sidebar (`app-sidebar.tsx` from task 13), and the route outlet.
- MUST rewrite `_app.tsx` to supply the authenticated app layout with the content column.
- MUST add route-level transition using `AnimatePresence` with `mode="wait"` and a 200ms fade (no scale, no layout-id).
- MUST honor `prefers-reduced-motion: reduce` — route transitions skip animation when set.
- MUST NOT break any existing route.
- MUST remove any obsolete layout code (old headers, old outlet wrappers) in the same PR.
- SHOULD verify every top-level route in `web/src/routes/_app/**` renders correctly after the rewrite.
</requirements>

## Subtasks

- [x] 14.1 Wrap the app in `<UIProvider>` at the main entry.
- [x] 14.2 Rewrite `__root.tsx` with sticky header + sidebar + outlet.
- [x] 14.3 Rewrite `_app.tsx` with the content column layout.
- [x] 14.4 Add `AnimatePresence` for route-level transitions (fade only).
- [x] 14.5 Delete obsolete layout fragments, old header components.
- [x] 14.6 Verify every top-level route renders and `make verify` passes.

## Implementation Details

TechSpec "Impact Analysis" flags `__root.tsx` + `_app.tsx` as modified. DESIGN.md §4 "Site Header (Marketing + Docs)" and §5 "Grid & Layout" document the visual expectations (operator UI mirrors the site header style with pinned blur on dark surfaces).

Route-level motion follows TanStack Router's integration pattern: key the outlet element by `location.pathname` and wrap in `AnimatePresence mode="wait"`.

### Relevant Files

- `web/src/main.tsx` — entry.
- `web/src/routes/__root.tsx` — root layout.
- `web/src/routes/_app.tsx` — authenticated layout.
- `web/src/components/app-sidebar.tsx` — consumed.
- `packages/ui/src/components/ui-provider.tsx` — UIProvider wrapper.
- `packages/ui/src/components/sidebar.tsx` + `split-pane.tsx` — layout primitives.
- **Design references** (read-only, do not edit):
  - `DESIGN.md §4 "Site Header" + §5 "Grid & Layout"` — sticky header + split-pane composition rules.
  - `docs/design/web-inspiration/src/app.jsx` — top-level router composition mock for operator shell.
  - `docs/design/design-system/ui_kits/docs/Shell.jsx` — docs-side shell reference (sticky header + sidebar tree + content column).
  - `docs/design/web-inspiration/styles/app.css` — keyframe + transition classes referenced by the mock (dot-pulse, transition-colors).

### Dependent Files

- Every route file under `web/src/routes/**`.
- All subsequent domain tasks (17+) render inside this shell.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-003: Adopt motion for UI animations](adrs/adr-003.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)

## Deliverables

- Rewritten `__root.tsx` + `_app.tsx`.
- `UIProvider` mounted at the app entry.
- Route-level fade transition via `AnimatePresence`.
- Unit tests with 80%+ coverage for any new helpers **(REQUIRED)**.
- Integration tests asserting every route renders inside the new shell **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] Sticky header renders wordmark + `ALPHA` chip + (placeholder) nav. *(`web/src/components/app-header.test.tsx` + `web/src/routes/-__root.test.tsx`)*
  - [x] Sidebar slot renders `AppSidebar`. *(`web/src/routes/-_app.test.tsx`)*
  - [x] Outlet renders the route element with `motion` key tied to pathname. *(`web/src/routes/-_app.test.tsx` "keys the motion shell by location pathname")*
  - [x] Under `prefers-reduced-motion: reduce`, `AnimatePresence` transitions fire with duration 0. *(`web/src/routes/-_app.test.tsx` "collapses route transitions to duration 0" + `resolveRouteTransitionDuration` unit)*
- Integration tests:
  - [x] Each top-level route renders inside the new shell — existing vitest suites under `web/src/routes/_app/**` all still pass (169 files / 1188 tests).
  - [x] Route fade transition firing — verified via the motion-mock `data-motion-duration`/`data-route-key` assertions in `_app.test.tsx`.
  - [x] Reduced-motion skips the animation — verified through `resolveRouteTransitionDuration(true) === 0` + mocked `useReducedMotionConfig` assertion.
  - [ ] Playwright visual baseline per route — deferred to **task 16** (`Wire Playwright visual snapshot baseline for web/`); the stable shell selectors (`app-header`, `app-content`, `app-route-motion`) land in this PR as the test hooks.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `UIProvider` wrapper visible at `web/src/main.tsx`.
- Every top-level route renders in the new shell without regressions.
- Playwright baseline snapshots committed for the shell + each top-level route's outer frame.
- `make verify` passes.

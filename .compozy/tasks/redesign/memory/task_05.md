# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Introduced `Sidebar` (rail + header + nav + footer slots, animated width collapse) and `SplitPane` (fixed-width list + flex detail, responsive stack with back nav) in `@agh/ui`.
- Deleted `web/src/components/ui/sidebar.tsx` + `use-sidebar{,-provider}` hooks; rewrote the ~4 local importers to consume `@agh/ui` directly or use plain tailwind-styled primitives.

## Important Decisions

- `Sidebar` owns the collapse trigger (button at the bottom of the rail). Keeps the trigger reachable even when `collapsed` is true and unifies `aria-expanded`/`data-state` semantics so host apps don't each re-invent the control. Controlled + uncontrolled modes both supported (`collapsed` + `onCollapse` or internal state from `defaultCollapsed`).
- Auto-collapse on narrow viewports is internal: the primitive tracks `useNarrowViewport(collapseBreakpoint)` and promotes that into `effectivelyCollapsed`. Keeps DESIGN.md's "rail stays visible; panel collapses on narrow viewports" rule enforced by the primitive itself, not every caller. `onCollapse` still only fires for user intent, not for viewport changes.
- Reduced-motion compliance is explicit: `duration = useReducedMotionConfig() ? 0 : 0.2s`. motion's built-in reducedMotion does not zero non-transform/opacity properties like `width`, so we short-circuit duration. Same pattern for `SplitPane` detail fade.
- `SplitPane` uses AnimatePresence with `mode="wait"` keyed on `hasDetail` so the empty state fades out before the real detail mounts. Kept it opacity-only so we don't fight the layout.
- On narrow viewports `SplitPane` hides the list when a detail is present and renders a `Back` button that delegates to `onDetailClose`. Kept the back action opt-in — narrow mode still works without it.

## Learnings

- `motion@12.38.0`'s `MotionConfig reducedMotion="always"` does NOT zero `width` animations; explicit `duration: 0` is required for width transitions to be instant under reduced-motion assertions.
- Under jsdom `matchMedia` is the only viewport signal; `useNarrowViewport` must (a) seed state from the initial MediaQueryList and (b) accept MediaQueryList-shaped objects in its change handler so the hook still works when only `matches` is updated via a manual _fire in tests.
- Motion sets `style.width` directly on the animated element (not via transforms), so asserting `panel.style.width` after a rerender is a valid way to test collapse-driven width changes when reduced motion is active.
- `web/src/components/ui/hooks/*` lived alongside the old shadcn sidebar — deleting the sidebar makes the hook directory orphan. Clean up both in the same pass; CI grep would otherwise catch it later.

## Files / Surfaces

- `packages/ui/src/components/sidebar.tsx` (new)
- `packages/ui/src/components/split-pane.tsx` (new)
- `packages/ui/src/components/sidebar.test.tsx` (new)
- `packages/ui/src/components/split-pane.test.tsx` (new)
- `packages/ui/src/components/stories/sidebar.stories.tsx` (new)
- `packages/ui/src/components/stories/split-pane.stories.tsx` (new)
- `packages/ui/src/index.ts` (exports)
- `web/src/components/ui/sidebar.tsx` (deleted)
- `web/src/components/ui/hooks/{use-sidebar,use-sidebar-provider}.ts` (deleted)
- `web/src/components/ui/stories/sidebar.stories.tsx` (deleted)
- `web/src/storybook/story-layout.tsx` (SidebarSurface composes `@agh/ui` Sidebar)
- `web/src/systems/agent/components/agent-sidebar-group.tsx` (rewritten against Collapsible + tailwind-only primitives)
- `web/src/systems/agent/components/agent-sidebar-group.test.tsx` (drops shadcn sidebar mocks)
- `web/src/systems/agent/components/stories/agent-sidebar-group.stories.tsx` (plain li/buttons)
- `web/src/systems/session/components/session-sidebar-item.tsx` (plain `<li><Link>` with existing data-testids preserved)
- `web/src/systems/session/components/session-sidebar-item.test.tsx` (drops shadcn sidebar mock)

## Errors / Corrections

- Pre-existing break-down confirmed: `web/src/systems/tasks/components/tasks-empty-state.test.tsx` fails 2 tests on `main` before this task (matches shared memory note about flaky web tests). `make verify` still fails at that test regardless of task 05 — no new regressions introduced.
- Pre-existing `packages/ui tsgo --noEmit` error on `accordion.test.tsx` (readonly array vs mutable Base UI `AccordionValue`) is also unchanged. Outside task 05 scope.

## Ready for Next Run

- Task 13 (app-sidebar rewrite) can compose `@agh/ui` `Sidebar` directly; slot API is `{ rail, header, nav, footer, collapsed, onCollapse, panelWidth, collapseBreakpoint, collapseLabel }`.
- Task 14 (root layout) and Tasks 17/20/23–27 consume `SplitPane` with props `{ list, detail, listWidth=340, detailEmpty, onDetailClose, narrowBreakpoint=768, backLabel }`.
- Task 28 (Workspace + Agent sidebar) can now properly rebuild `AgentSidebarGroup`/`SessionSidebarItem` — the temporary tailwind classes landed here are load-bearing only until that task.

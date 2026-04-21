# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Rewrote `web/src/components/app-sidebar.tsx` as a thin composition over `@agh/ui` `Sidebar`, using its `rail | header | nav | footer` slots and its built-in collapse trigger.
- Removed the host-owned collapse buttons (`collapse-toggle`, `expand-toggle`); the Sidebar primitive's own button owns that affordance now.
- Consumed `@agh/ui` primitives: `Sidebar`, `Collapsible/Trigger/Content`, `ConnectionIndicator`, `StatusDot`, `Kbd`, `cn`, and the `ConnectionStatus` type. No imports from the deleted `web/src/components/ui/` tree.

## Important Decisions

- Renamed the host prop `onToggleCollapsed` → `onCollapseChange(next: boolean)` to match `Sidebar.onCollapse(next)` directly, and updated `useAppLayout` to return `setCollapsed` (via `useSidebarStore(state => state.setCollapsed)`) instead of `toggle`. Simpler end-to-end: store → hook → component, no boolean inversion in between.
- Session state → StatusDot mapping (sidebar scope only): `active → success`, `starting → warning pulse`, `stopping → neutral pulse`, `stopped → neutral`.
- Wordmark slot renders `agh` in `font-wordmark` + an `ALPHA` chip (mono 9px, uppercase, muted border) inside the header, matching DESIGN.md §4 "Site Header" lockup so the operator and marketing surfaces share the same wordmark treatment.
- Workspace rail uses square-rounded app-logo, circular workspace avatars with `data-active={true/false}` for test assertion (no more `aria-checked` ambiguity), and a dashed `+` affordance that remains visible even when `workspaces=[]`.
- Kept `SearchInput` out of the nav slot for now — the nav shows a static search *placeholder row* with `⌘K` kbd hint. Live search wiring is not in scope for task 13 (no upstream command palette yet; follow-up if a later task adds one).

## Learnings

- `@agh/ui` `Sidebar` needs no Host-owned expand button; omitting it keeps visual hierarchy clean and matches the primitive's aria contract (`aria-expanded` flips on its own trigger).
- In vitest, using `UIProvider reducedMotion="always"` around the AppSidebar short-circuits the motion width animation so the initial render is stable and jsdom assertions don't race the transition.
- The Sidebar primitive's built-in `useNarrowViewport` hook auto-collapses below 768px. Because the test-setup `matchMedia` mock always returns `matches: false`, the unit tests never trip that path. Worth remembering if a future test explicitly needs to exercise narrow auto-collapse.
- `@tanstack/react-router` `Link` with `params={{ id: session.id }}` renders `/session/$id` literally unless the test mock substitutes the `$param` — the new AppSidebar test's Link mock handles that substitution so the href check works without pulling in the real router.

## Files / Surfaces

- `web/src/components/app-sidebar.tsx` — full rewrite (sidebar chrome is now `@agh/ui`; only local domain content stays here).
- `web/src/components/app-sidebar.test.tsx` — full rewrite; uses real `@agh/ui` primitives under `UIProvider reducedMotion="always"`, only `@tanstack/react-router` is mocked.
- `web/src/components/stories/app-sidebar.stories.tsx` — new; `Default`, `Collapsed`, `NoWorkspaces`, `ManyWorkspaces`, `Disconnected`, `Reconnecting`, plus `TogglesCollapse` + `SwitchesWorkspace` `play()` interaction stories.
- `web/src/hooks/routes/use-app-layout.ts` — returns `setCollapsed` instead of `toggleCollapsed`.
- `web/src/routes/_app.tsx` — passes `collapsed` + `onCollapseChange={page.setCollapsed}` to `AppSidebar`.
- `web/src/routes/-_app.test.tsx` — extended the `@/stores/sidebar-store` mock to include `setCollapsed`.

## Errors / Corrections

- First pass of the full vitest web suite briefly failed while `oxfmt` reformatted my files; re-ran after the auto-format and the full suite went green (168 files, 1180 tests).
- During iterative verification I did a `git stash --include-untracked` before running `make lint` on the base — the stash pop flagged "local changes would be overwritten" because the pre-existing `internal/observe/tasks.go` modifications were still in the stash. Recovered by `git checkout HEAD -- internal/observe/tasks.go internal/observe/tasks_test.go internal/store/globaldb/global_db_task_test.go` then `git stash pop`. Lesson for future runs: don't use stash as a "test on a clean base" shortcut when the base already has unrelated pre-existing modifications documented as Open Risks.

## Ready for Next Run

- Task 14 (root layout + route motion) composes this `AppSidebar` at the shell level and will mount `UIProvider` in `web/src/main.tsx`. The sidebar is now fully controlled (`collapsed` + `onCollapseChange`), so the shell can share sidebar state with other chrome (command palette, responsive burger) by reading `useSidebarStore` directly if needed.
- No new `@/components/ui/*` imports exist in `web/src/components/app-sidebar.tsx`; consolidation rule holds.

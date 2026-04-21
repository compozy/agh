# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite the root + authenticated route shells so the app boots under `<UIProvider reducedMotion="user">`, renders a DESIGN.md §4-style sticky blurred header, keeps the sidebar rooted in `_app.tsx`, and applies a 200ms `AnimatePresence mode="wait"` fade between routes (0ms under prefers-reduced-motion).

## Important Decisions

- **Global sticky header lives in `__root.tsx`** (new `web/src/components/app-header.tsx`). The sidebar stays in `_app.tsx` because it depends on workspace context. Task spec's "root renders sidebar" phrasing is interpreted as "the root-level shell includes the sidebar surface" — the sidebar is physically rendered by `_app.tsx` underneath the global header.
- **Wordmark ownership moved from sidebar header → global app header.** DESIGN.md §4 allows only one wordmark per surface; task 13 had temporarily placed it in the sidebar header because no global header existed yet. Task 14's 14.5 (delete obsolete layout fragments) covers the move. The sidebar header now surfaces the active workspace name + search affordance, matching `docs/design/web-inspiration/src/sidebar.jsx`.
- **Dropped `ThemeProvider` + `next-themes` entirely.** DESIGN.md locks the app to dark mode only; the provider had no consumer. Removed from `web/package.json` via `bun remove next-themes`.
- **Route motion duration is extracted to `resolveRouteTransitionDuration(reducedMotion)` + exported from `_app.tsx`.** Lets tests assert the 0.2 ↔ 0 gating without reaching into mocked motion internals. The helper is the sanctioned pattern for future route-transition tweaks.
- **Motion mocked in `_app.test.tsx` via `data-motion-*` attributes.** `AnimatePresence` renders a passthrough div exposing `mode`/`initial`; `motion.div` passthrough serializes `transition`/`initial`/`animate`/`exit` onto `data-motion-*`. This pattern scales for later route-transition tests without coupling to framer's internal DOM.
- **`_app` test mock introduces `useLocation` with `select` support.** The mock returns a mutable `mockPathname` so tests can swap routes by re-rendering.
- **Sidebar `HeaderSlot` reduced to workspace name + search icon.** Kept search stub for DESIGN.md §4 docs-shell parity; a real search trigger is a future task.

## Learnings

- TanStack Router route files can safely export additional named values alongside `Route` — `routeTree.gen.ts` only imports the `Route` symbol.
- `@testing-library/react`'s `rerender()` cannot be called after `unmount()`; for route-swap tests use two separate `render()` calls with explicit `first.unmount()` between them.
- `vi.fn<[], boolean>()` is legacy syntax under current vitest; the new form is `vi.fn<() => boolean>()`. The legacy shape still *runs* but yields a never-typed mock that fails strict typecheck inside `tsgo --noEmit`.
- Every consumer of motion-animated routing must call `useReducedMotionConfig()` (not plain `useReducedMotion()`) to respect the `<UIProvider>` test harness — same rule already documented in shared memory, now applied at the route level.

## Files / Surfaces

- Added: `web/src/components/app-header.tsx` + `app-header.test.tsx`.
- Rewritten: `web/src/main.tsx`, `web/src/routes/__root.tsx`, `web/src/routes/_app.tsx`, `web/src/routes/-__root.test.tsx`, `web/src/routes/-_app.test.tsx`.
- Trimmed: `web/src/components/app-sidebar.tsx` (`HeaderSlot` no longer carries wordmark + ALPHA chip), `web/src/components/app-sidebar.test.tsx` (Header block now asserts workspace-name + wordmark-removed), `web/src/components/stories/app-sidebar.stories.tsx` (docblock updated).
- Dependency: removed `next-themes` from `web/package.json` (no remaining consumer).

## Errors / Corrections

- Initial `_app.test.tsx` route-swap case used `rerender()` after `unmount()` → `"Cannot update an unmounted root"`. Fixed by doing two independent `render()` calls.
- First `vi.fn<[], boolean>()` broke `tsgo --noEmit`. Switched to `vi.fn<() => boolean>()`.

## Ready for Next Run

- Task 16 (Playwright visual baselines for `web/`) should key snapshots off the new shell: sticky header + sidebar + content outlet. Baseline every top-level `_app/*` route plus `/design-system`. The `data-testid="app-header"` / `"app-content"` / `"app-route-motion"` selectors are stable and motion-free under `prefers-reduced-motion: reduce`.
- Future task (tracked by task 16 or later): verify in a live dev server that navigating between two `_app/*` routes shows the 200ms fade (unit-level mock only asserts the motion props; visual regression owns the render evidence).
- Shared memory update promoted: route-level motion contract (helper + mock pattern + duration) and the wordmark-ownership shift.

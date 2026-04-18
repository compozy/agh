# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Turn the observer-backed dashboard and inbox aggregate read models into first-class `/_app/tasks` surfaces that sit alongside the existing list/kanban modes.
- Keep route-level orchestration inside `useTasksPage` plus the tasks system barrel; components remain presentational.

## Important Decisions
- Mode switching stays in React state on `useTasksPage` (no search params) because no sibling route uses URL-driven search params today; adding TanStack Router search param plumbing would expand scope and is not required by the task.
- Dashboard uses a single `TasksDashboardView` shell that composes four section components (cards, status breakdown, queue/health, active runs). This keeps each section independently testable and reusable without recreating the Paper layout in one giant component.
- Inbox keeps the aggregate groups as-is from the backend (no client-side re-bucketing) — the lane tabs only drive the filter query; the grouping headers within the body render whatever the server returned.
- Lane filter supports an explicit "All" tab to preserve the default dashboard-wide view from the Paper exports, even though the backend filter is `lane | undefined` on the wire.
- The create button is hidden on the inbox mode because Paper swaps "+ Task" for "Archive all"; since archive-all is not a first-class command today, we simply collapse the button until that surface exists.
- Component test helpers live in `web/src/systems/tasks/components/test-fixtures.ts` (non-test `.ts` so vitest ignores it) to avoid repeating large payload builders across component + route tests.

## Learnings
- `InboxLaneFilter` is exported from `use-tasks-page.ts` so inbox components can type against the route hook without triggering a circular barrel import — the tasks system barrel now re-exports inbox components but the components themselves import the filter type from the route hook directly.
- Route-level integration tests can mock `@tanstack/react-router` `Link` to a plain anchor (mirroring `-tasks.test.tsx`), which keeps dashboard/inbox tests lightweight without a full memory router.
- `formatDurationMs` + `formatPercent` belong in the tasks system formatters so every dashboard/inbox section can format backend durations and percentages identically; avoided pulling in a date-fns dependency for this scope.

## Files / Surfaces
- `web/src/hooks/routes/use-tasks-page.ts` — adds inbox lane/unread/search state plus approve/reject/archive/dismiss/mark-read/retry handlers and exposes dashboard + inbox loading/error.
- `web/src/routes/_app/tasks.tsx` — adds Dashboard and Inbox pills, hides the create button on inbox mode, and renders the new aggregate views.
- `web/src/systems/tasks/components/tasks-dashboard-*.tsx` — new dashboard sections and view shell.
- `web/src/systems/tasks/components/tasks-inbox-*.tsx` — new inbox lane tabs, item card, and view shell.
- `web/src/systems/tasks/components/test-fixtures.ts` — shared `TaskDashboardView` / `TaskInboxView` / `TaskInboxItem` fixture builders used by view + route tests.
- `web/src/systems/tasks/lib/task-formatters.ts` — gains `formatDurationMs` / `formatPercent` helpers.
- `web/src/systems/tasks/index.ts` — re-exports the new components and formatter helpers.
- Tests: `tasks-dashboard-cards.test.tsx`, `tasks-dashboard-view.test.tsx`, `tasks-inbox-view.test.tsx`, expanded `use-tasks-page.test.tsx` and `-tasks.test.tsx`, plus additional `task-formatters.test.ts` coverage for the new helpers.

## Errors / Corrections
- Initial `tasks-dashboard-view.test.tsx` wrapped components in a fake TanStack Router + QueryClient tree; the tree broke because we were trying to place children inside a RouterProvider. Switched to mocking `@tanstack/react-router` `Link` to a plain anchor, matching the other route tests.

## Ready for Next Run
- Dashboard + Inbox surfaces live under `/_app/tasks` as first-class modes with loading/error/empty states and >80% coverage on the new components + route hook paths.
- `make web-lint`, `make web-typecheck`, and `bun run test` all pass cleanly.
- Follow-up task_17 (multi-agent live) can reuse `TasksDashboardActiveRuns` for live run rendering if desired; otherwise the task is scoped to the live tree/session drill-down and doesn't touch the new aggregates.
- Follow-up: when a first-class "Archive all" command lands, wire it into the inbox mode meta slot that currently collapses the create button.

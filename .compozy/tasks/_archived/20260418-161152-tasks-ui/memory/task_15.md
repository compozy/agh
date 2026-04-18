# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace the placeholder `/tasks/$id` and `/tasks/$id/runs/$runId` routes with task-native detail and run-detail experiences backed by the task-detail/timeline/tree/run-detail reads already provided by the tasks system scaffold.
- Keep session context as drill-down only; the task live model is the primary source.

## Important Decisions
- Extended `useTaskDetailPage` with timeline pagination (`timelineLimit` + `handleTimelineLoadMore`), cancel/publish/enqueue mutations, and `isLive`/`isTimelineSaturated` derivations; preserved the existing shape so the scaffold test still passes.
- Extended `useTaskRunPage` with `handleCancelRun` (wraps `useCancelTaskRun`) and `isLive`; linked-session drill-down is done via `Link to="/session/$id"` and a sidebar identity row.
- Tabs live in a new `TasksDetailTabs` component (overview/runs/timeline/children/dependencies) with counts and a Live pill; panel switching is state-only, URL sync is not required by this task.
- Timeline component derives human messages from event type with a fallback to `payload.message`; failed events use danger tone, in-progress events show a Live pill.
- Run-detail layout is two-column at `xl+` (main activity + session drill-down, right-side identity + progress panels).
- Max attempts are only shown on the task-detail run chip (from `summary.active_run.max_attempts`); run-detail view only exposes the run record which does not carry `max_attempts`, so we intentionally omit "of N".

## Learnings
- `getTaskRun` payload’s inner `task` view is a thin summary — it lacks `max_attempts`; only `getTask` (task detail) carries that field, and only the `summary.active_run` chip has a run-scoped max.
- `useChildMatches` is the established pattern for the tasks route tree; the detail route must only render its own content when there are no child matches, otherwise it forwards to `<Outlet />` for run-detail nesting.
- React + children prop: renamed the child-panel prop to `items` to avoid shadowing the reserved React `children` prop while still iterating child tasks.

## Files / Surfaces
- Routes: `web/src/routes/_app/tasks.$id.tsx`, `web/src/routes/_app/tasks.$id.runs.$runId.tsx`
- Route hooks: `web/src/hooks/routes/use-task-detail-page.ts`, `web/src/hooks/routes/use-task-run-page.ts`
- New tasks-system components: `tasks-detail-header`, `tasks-detail-tabs`, `tasks-detail-overview-panel`, `tasks-timeline-panel`, `tasks-detail-runs-panel`, `tasks-detail-children-panel`, `tasks-detail-dependencies-panel`, `task-run-detail-header`, `task-run-detail-panels` (identity/progress/activity), `task-run-detail-session-link`
- Barrel updated: `web/src/systems/tasks/index.ts`
- Tests: component tests for each new component, updated `-tasks.$id.test.tsx` and `-tasks.$id.runs.$runId.test.tsx`, extended hook tests with new state transitions.

## Errors / Corrections
- First attempt referenced `record.max_attempts` in run-detail views — removed after typecheck confirmed the run schema omits it.
- Initial draft had an early-return before a `useMemo` — replaced with a plain `const tabItems` array to keep render stable without violating hooks rules.

## Ready for Next Run
- Task_16 (dashboard + inbox routes) will continue from this scaffold; detail and run-detail deep links are stable, but a future polish pass may want to wire URL sync for the active panel param and add an SSE hook around `/api/tasks/{id}/stream` (not required for task_15).
- Task_17 (multi-agent live) can link into these routes and reuse `TasksTimelinePanel` for the unified timeline if needed.

# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Reusable `web/src/systems/tasks` scaffold (adapter, query infra, hooks, formatters, public barrel) + route-hook orchestration for base/detail/run-detail so tasks 14-17 can focus purely on UI behavior.

## Important Decisions
- Kept the pre-existing adapter/query-keys/query-options from scratch as-is; only extended the barrel and layered hooks on top. Avoids rewriting surfaces that already match the techspec.
- Query key taxonomy: `tasks.all` is the only root, with sibling namespaces `list/detail/runs/timeline/tree/run-detail/dashboard/inbox/triage`. Mutation invalidation fans out via root keys (`tasks.runsRoot()`, etc.) so list, detail, live, and aggregate reads stay coherent after any write.
- Split hooks into domain-flavored files (`use-tasks`, `use-task-live`, `use-task-dashboard`, `use-task-inbox`, `use-task-actions`) instead of a monolithic `use-tasks.ts`, matching the techspec's explicit hook list.
- Adapter tests use `mockJsonSequence` (mockImplementation returning a new Response per call) to avoid the "Body already read" error when one test exercises multiple lifecycle endpoints. This is the right pattern to reuse for any future multi-call adapter test.
- Route hooks stay presentational-friendly: no optimistic UI, no toast wiring, no URL-state escape hatches. They expose flat props that task_14/15/16 can consume to drive screen components. URL/search-param plumbing is deferred until a screen actually needs it.

## Learnings
- `TaskDetailView` is the `task + summary + children + dependencies` envelope; `TaskRunDetailView` is the `run + session + summary + task` envelope. Tests that want `.id` had to go through `.task.id` / `.run.id` / etc.
- `useTask` returns `TaskDetailView`, so test fixtures must match that shape rather than the flat `TaskRecord` shape — caught by tsgo during typecheck.
- oxfmt auto-rewrites test files to collapse readable multi-line arrays; don't fight it.

## Files / Surfaces
- `web/src/systems/tasks/hooks/use-tasks.ts`, `use-task-live.ts`, `use-task-dashboard.ts`, `use-task-inbox.ts`, `use-task-actions.ts` (+ matching tests)
- `web/src/systems/tasks/lib/task-formatters.ts` (+ test)
- `web/src/systems/tasks/lib/query-keys.test.ts`, `query-options.test.ts`
- `web/src/systems/tasks/adapters/tasks-api.test.ts`
- `web/src/systems/tasks/index.ts` (public barrel re-exports everything tasks 14-17 will need)
- `web/src/hooks/routes/use-tasks-page.ts`, `use-task-detail-page.ts`, `use-task-run-page.ts` (+ tests)

## Errors / Corrections
- Initial adapter test reused a single `mockJsonResponse` for multi-call lifecycle tests → `Body is unusable: Body has already been read`. Fixed by introducing `mockJsonSequence` (mockImplementation returning a fresh Response each call).
- First version of `useTask` test asserted `result.current.data?.id`; `getTask` returns `TaskDetailView` (nested under `.task`), so tsgo rejected the access. Updated fixture to `{ task, summary: {} }` and assert on `.task.id`.

## Ready for Next Run
- task_14/15/16 can compose `useTasksPage` / `useTaskDetailPage` / `useTaskRunPage` alongside the exported formatters and hooks without reaching into adapters. Add dialog/drawer/draft state locally inside their page hooks or dedicated stores — the scaffold intentionally stays free of UI-specific state so screen work stays focused.
- `useTasksPage` currently leaves dashboard/inbox filter shaping open (it only forwards scope + workspace). If a dashboard screen needs origin_kind / network_channel / lane / unread toggles, extend the hook's filters rather than duplicating them per screen.

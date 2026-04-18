# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- First-class `/_app/tasks` route family + shared shell + Tasks sidebar nav, thin route files ready for task_13+ orchestration.

## Important Decisions
- Flat-file nested routing: `tasks.tsx` is a layout with `<Outlet />`; `tasks.$id.tsx` and `tasks.$id.runs.$runId.tsx` render inside via TanStack auto-nesting. Landing vs Outlet chosen with `useChildMatches()` so the shared shell DOM survives navigation between base, detail, and run-detail.
- `TasksPageShell` is a thin wrapper around the existing `WorkspacePageShell` (title "Tasks", `ListChecks` icon). Keeps visual parity with `network`/`knowledge`/`automation` shells and leaves later tasks free to layer controls/meta/body content.
- Sidebar `NavItem` gained an optional `fuzzy` flag; Tasks uses it so the nav stays active across `/tasks`, `/tasks/$id`, and `/tasks/$id/runs/$runId`. Other entries keep strict match semantics (no behavior change to their existing tests).

## Learnings
- `bun run dev:raw` regenerates `web/src/routeTree.gen.ts` via the TanStack router Vite plugin; after I hand-authored the file, running vite briefly and re-reading confirmed the generator produced the same structure (including the nested `AppTasksIdRouteWithChildren` / `AppTasksRouteWithChildren` composition).
- Vitest does not run the router plugin; keeping `routeTree.gen.ts` in sync is manual unless a vite command runs first.
- `make web-build` (vite build + tsgo) emits separate chunks for `tasks`, `tasks._id`, and `tasks._id.runs._runId`, confirming auto code-splitting is working for the new routes.

## Files / Surfaces
- `web/src/systems/tasks/components/tasks-page-shell.tsx` + `web/src/systems/tasks/index.ts`
- `web/src/routes/_app/tasks.tsx`, `tasks.$id.tsx`, `tasks.$id.runs.$runId.tsx`
- `web/src/components/app-sidebar.tsx` (Tasks nav entry + `fuzzy` option on NavItem)
- `web/src/routeTree.gen.ts` (regenerated)
- Tests: `web/src/components/app-sidebar.test.tsx`, `web/src/routes/_app/-tasks*.test.tsx`, `web/src/routes/_app/-tasks.router.integration.test.tsx`, `web/src/systems/tasks/components/tasks-page-shell.test.tsx`

## Errors / Corrections
- Initial draft wanted to force `fuzzy: true` on every sidebar nav item; caught that this would loosen the existing strict active-state tests for `/automation`, `/bridges`, etc. Reverted to an opt-in `fuzzy` prop so only Tasks uses fuzzy matching.

## Ready for Next Run
- task_13 (`web/src/systems/tasks` scaffold) can now land adapters/hooks/components under the existing `systems/tasks/` barrel without touching global navigation or route taxonomy. The routes intentionally render placeholders; screen orchestration belongs to task_13/14/15/16.

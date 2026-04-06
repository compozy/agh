# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `workspaces` persistence to the global SQLite DB, move session rows from bare workspace path strings to `workspace_id`, implement `workspace.WorkspaceStore` on `store.GlobalDB`, and cover the new behavior with unit tests.

## Important Decisions
- Use a real `sessions.workspace_id -> workspaces.id` foreign key in the global schema to align with the task requirement for FK stability.
- Keep workspace store behavior inside `internal/store` only; resolver/name auto-dedup logic remains out of scope for this task.
- Auto-migrate legacy global DBs that still have `sessions.workspace` by backfilling `workspaces` rows, copying session/link tables into the new schema, and swapping tables inside one transaction.
- Stage `WorkspaceID` through `store`, `session`, and `observe` metadata surfaces now so task_04 can inject the resolver without another schema rename.

## Learnings
- Task 01 already added `internal/workspace` types and errors, and its workflow artifacts are untracked.
- Existing runtime sessions still need both the workspace path and workspace ID for now: the path remains in session metadata/runtime config loading, while the global registry now persists only `workspace_id`.
- `make verify` initially failed on `staticcheck` because tests passed literal nil contexts; replacing those with typed `var nilCtx context.Context` restored compliance without weakening validation.

## Files / Surfaces
- `internal/store/schema.go`
- `internal/store/global_db.go`
- `internal/store/global_db_test.go`
- `internal/store/store.go`
- `internal/session/session.go`
- `internal/session/manager.go`
- `internal/session/query.go`
- `internal/observe/observer.go`
- `internal/observe/reconcile.go`
- `internal/observe/helpers_test.go`
- `internal/observe/observer_test.go`
- `internal/observe/reconcile_test.go`
- `internal/store/meta_test.go`
- `internal/store/store_helpers_test.go`

## Errors / Corrections
- Fixed the final `make verify` failure by replacing nil-context test calls with typed nil context variables to satisfy `staticcheck`.

## Ready for Next Run
- Implementation, focused tests, coverage check, and `make verify` are complete. Remaining work is bookkeeping only: update task tracking, keep workflow artifacts out of the commit, and create one local code-only commit.

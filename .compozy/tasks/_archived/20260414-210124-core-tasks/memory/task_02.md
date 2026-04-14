# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Persist `internal/task` `Task` and `TaskRun` records in `internal/store/globaldb` with schema, CRUD/list/query support, and tests that satisfy the task_02 coverage/integration requirements.

## Important Decisions
- Added task/run schema directly to `globaldb` bootstrap statements instead of a standalone migration because these are new tables with no legacy shape to transform.
- Implemented task/run persistence in a dedicated `internal/store/globaldb/global_db_task.go` file and kept task_03 surfaces (`dependencies`, `events`, idempotency store) out of scope.
- Reused `internal/task` validation and immutable-field helpers for canonical shape enforcement, then added store-level reference prechecks for missing workspace, parent task, and run task ids.
- `UpdateTaskRun` enforces single-assignment for `session_id` once a run is already bound.

## Learnings
- Package coverage for `internal/store/globaldb` cleared the 80% target after adding focused tests for not-found/reference-error branches and normalization defaults; broad happy-path tests alone were not enough.
- Stable ordering assertions for task list limits require distinct timestamps because the list query orders by `updated_at`, `created_at`, then `id`.

## Files / Surfaces
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/store/globaldb/global_db_task_integration_test.go`

## Errors / Corrections
- Initial `ListTasks(limit)` test assumed a deterministic order while all fixture timestamps were identical; fixed by assigning distinct timestamps to the filter fixtures before re-running tests.

## Ready for Next Run
- Task tracking still needs to be updated after verification/self-review, and the automatic local commit should stage only code files, not workflow memory or `.compozy` tracking files.

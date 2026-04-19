# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement detached harness work on the existing `task` / `task_run` runtime so harness-owned async work is durable, inspectable, and boot-recoverable without introducing a parallel background-run entity.

## Important Decisions
- Keep the harness mapping daemon-owned in `internal/daemon/harness_detached_work.go`; prompt-facing code does not construct task records directly.
- Persist harness-specific run data in generic task substrate fields:
  - `task.Task.Metadata` carries detached harness owner/session/wake-target scope metadata.
  - `task.Run.Metadata` is persisted in `task_runs.metadata_json` for run-level targeting and recovery.
- Reuse task-runtime idempotency by deriving deterministic detached task IDs and matching duplicate submissions against both task-level and run-level detached metadata before reusing a queued run.

## Learnings
- Adding `task_run` metadata to the API contract requires regenerating `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`; `make test-integration` will fail `codegen-check` until those artifacts are refreshed.
- The Teams bridge initialize path reports initial state from a background goroutine; tests that mutate provider seams after `initialize` must synchronize on the existing state-report marker first or the race detector will flag the test.

## Files / Surfaces
- `internal/daemon/harness_detached_work.go`
- `internal/daemon/task_runtime.go`
- `internal/daemon/task_runtime_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/task/types.go`
- `internal/task/validate.go`
- `internal/task/interfaces.go`
- `internal/task/manager.go`
- `internal/task/manager_test.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/api/contract/tasks.go`
- `internal/api/core/tasks.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `extensions/bridges/teams/provider_test.go`

## Errors / Corrections
- Targeted Go tests initially failed because two `task_runs` read paths still selected the pre-metadata column set; `GetTaskRunByIdempotencyKey` and `getTaskRunWithExecutor` were updated to include `metadata_json`.
- The first full `make verify` run failed on a pre-existing race in `extensions/bridges/teams/provider_test.go`; the fix was to wait for the initial state-report marker before mutating `runtime.apiFactory`, matching the provider's actual async initialize contract.

## Ready for Next Run
- Task 07 can consume detached harness completions from normal task/run persistence by reading the daemon-owned detached metadata instead of inventing a second completion substrate.
- Verification evidence for this task: targeted package tests, detached-runtime integration tests, `make test-integration`, and `make verify` all passed on 2026-04-18.

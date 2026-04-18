# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_06 as real backend inbox capability: observer-backed lane grouping plus manager-owned approval and triage mutations, with unit/integration coverage and later tracking/commit updates only after clean verification.

## Important Decisions
- Lane precedence is singular and deterministic: `archived -> approvals -> failed_runs -> blocked -> my_work`.
- `dismiss` is actor-scoped and activity-sensitive, not a permanent archive. A dismissed item is hidden only until a newer task/run/event activity timestamp makes it visible and unread again.
- `archive` is actor-scoped and durable. Archived items stay in the archived lane and do not contribute to unread totals.
- Approval decisions are manager-owned task mutations. Only manual-approval tasks in `pending` state may be approved or rejected; other approval actions return a transition conflict.
- Lane-filtered inbox queries must filter totals and grouped counts to the selected lane instead of reporting global totals.

## Learnings
- Durable triage persistence from task_02 is already sufficient for the write side; the missing pieces are manager command methods, actor-scoped triage listing, and the observer inbox aggregate.
- `internal/api/contract/tasks.go` already defines the public inbox/triage vocabulary from task_07, so task_06 should shape backend models to match that language without editing transports.
- Expanding `observe.Registry` ripples into daemon test doubles; `internal/daemon/daemon_test.go` needed the new `ListTaskTriageStates` stub to keep the repo verification gate green.
- The repository verification gate caught two observer hygiene issues after the first implementation pass: `taskInboxFromSnapshot` exceeded the function-length limit and the priority comparator exceeded the line-length limit.

## Files / Surfaces
- `internal/daemon/daemon_test.go`
- `internal/task/interfaces.go`
- `internal/task/types.go`
- `internal/task/manager.go`
- `internal/task/manager_test.go`
- `internal/task/manager_integration_test.go`
- `internal/task/interfaces_integration_test.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/store/globaldb/global_db_task_integration_test.go`
- `internal/observe/observer.go`
- `internal/observe/tasks.go`
- `internal/observe/tasks_test.go`
- `internal/observe/tasks_integration_test.go`

## Errors / Corrections
- `brainstorming` would normally trigger for a new feature, but this run is using the already-approved PRD/TechSpec/ADR design under `cy-execute-task`; a second design loop would conflict with the task workflow.
- The first inbox aggregation pass incorrectly preserved global counts for lane-filtered queries; the observer now applies the lane filter before incrementing totals and lane counts.

## Ready for Next Run
- Verification evidence for completion:
  - `go test ./internal/task -coverprofile=/tmp/task06_task.cover && go tool cover -func=/tmp/task06_task.cover | tail -n 8` -> 80.3%
  - `go test ./internal/observe -coverprofile=/tmp/task06_observe.cover && go tool cover -func=/tmp/task06_observe.cover | tail -n 8` -> 82.6%
  - `go test ./internal/store/globaldb -coverprofile=/tmp/task06_store.cover && go tool cover -func=/tmp/task06_store.cover | tail -n 8` -> 80.5%
  - `go test -tags integration ./internal/task ./internal/observe ./internal/store/globaldb` -> pass
  - `make verify` -> pass after fixing daemon registry stub and observer lint findings

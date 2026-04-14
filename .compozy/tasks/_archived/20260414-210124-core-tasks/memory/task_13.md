# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add task-domain read-side observability in `internal/observe` so operators can inspect queue depth, stuck runs, task/run totals, ownership/channel/origin breakdowns, and recovery/cancellation outcomes without creating a second lifecycle authority.

## Important Decisions
- Use the approved PRD/TechSpec/ADR set as the design baseline for this implementation task instead of running a separate design approval loop.
- Prefer read-only aggregation over durable `tasks`, `task_runs`, `task_events`, and `network_audit_log` records rather than introducing new authoritative projection tables.
- Add task observability as exported `internal/observe` query and health surfaces (`QueryTaskSummary`, `QueryTaskMetrics`, `Health.Tasks`) instead of extending transport contracts in the same task.
- Use configurable stuck thresholds on the observer with default windows of 5m for `claimed`/`starting` and 30m for `running`.

## Learnings
- `internal/observe` is still session/bridge-centric; it has no task-specific query model or health block yet.
- Task lifecycle already emits durable audit events for enqueue/claim/start/complete/fail/cancel/force-stop/recovery/rejection in `internal/task`, and network task ingress already writes normalized accepted/rejected audit rows in `network_audit_log`.
- Duplicate network ingress can be derived read-side by comparing accepted `task.run.enqueue` audit rows against canonical network-origin `task.run_enqueued` task events, clamping the mismatch at zero.
- Recovery outcomes needed by observe are already encoded in `task.run_recovered` event payloads, so no extra persistence shape was required.

## Files / Surfaces
- `internal/observe/observer.go`
- `internal/observe/query.go`
- `internal/observe/health.go`
- `internal/observe/reconcile.go`
- `internal/observe/tasks.go`
- `internal/observe/tasks_test.go`
- `internal/observe/tasks_integration_test.go`
- `internal/store/globaldb/global_db_observe.go`
- `internal/store/globaldb/global_db_network_audit.go`
- `internal/task/manager.go`
- `internal/network/tasks.go`
- `internal/daemon/daemon_test.go`

## Errors / Corrections
- `make verify` initially failed on an unused helper type in `internal/observe/tasks.go`; removing the dead type cleared lint without changing behavior.

## Ready for Next Run
- Task 13 is complete. Task tracking files were updated in the working tree, and the task-owned source/test changes were committed as `778f663` (`feat: add task observe projections`).
- A final post-commit `make verify` pass also succeeded against the current tree.
- If follow-on API work needs task health externally, transport-layer mapping from `internal/observe.Health.Tasks` remains the next logical seam.

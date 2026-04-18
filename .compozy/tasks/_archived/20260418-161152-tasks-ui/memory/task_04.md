# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement backend-owned task live surfaces for timeline, replayable stream, task-tree status, and run-detail reads without introducing a parallel event store.

## Important Decisions
- Reused persisted `task_events` as the single live history source and used SQLite `task_events.rowid` as the stable sequence for replay and reconnect semantics.
- Kept the task manager/service as the owner of task-history joins, descendant aggregation, and run/session enrichment so later API tasks can expose one task-native contract.
- Made runtime/session usage enrichment optional through an injected `RuntimeViewReader`; run-detail payloads stay valid when session telemetry is unavailable.
- Root task streams fan out descendant task events and use `after_sequence` replay instead of guaranteeing lossless live delivery from notifier channels alone.

## Learnings
- Expanding `task.Store` with sequence-backed event reads affects boot-time daemon tests because the daemon task runtime only initializes when its registry satisfies the full task store contract.
- The repo-wide `make verify` tail exposed that compatibility gap even after the new task-focused tests were already green.

## Files / Surfaces
- `internal/task/live.go`
- `internal/task/live_types.go`
- `internal/task/interfaces.go`
- `internal/task/manager.go`
- `internal/task/manager_test.go`
- `internal/task/manager_integration_test.go`
- `internal/task/interfaces_integration_test.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/daemon/daemon_test.go`

## Errors / Corrections
- `make verify` initially failed in `internal/daemon` because `recordingRegistry` no longer satisfied the expanded task store surface; added no-op `GetTaskEventRecord` and `ListTaskEventRecords` methods to keep task runtime boot coverage intact.

## Ready for Next Run
- Task live reads are verified and tracking is ready for downstream API contract/transport work in tasks 07-09.

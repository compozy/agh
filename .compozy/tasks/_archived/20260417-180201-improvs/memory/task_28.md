# Task Memory: task_28.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute the report-first improvements pass for `internal/store/`: build all mandatory inventories, add benchmark coverage for selected hot paths, land any justified package-local fixes, and finish with clean `make verify` evidence plus task/report updates.

## Important Decisions
- Keep production edits strictly inside `internal/store/`; unrelated `internal/session` and task-tracking changes remain untouched.
- Treat the missing `.compozy/tasks/improvs/adrs/` directory as “no ADR context provided” rather than a blocking conflict.
- Use the established workflow-memory convention that `ubs` becomes `not-run` if no callable skill runner exists in this session.
- Keep tracking-only files out of the automatic deliverable commit even though task status still needs to be updated locally.

## Learnings
- `internal/store` runtime concurrency is concentrated in `sessiondb.SessionDB`: one owned writer goroutine, three package-local channels, one `sync.RWMutex`, and all production `select` sites are cancellation-aware.
- The largest production files are `global_db_automation.go`, `global_db_bridge.go`, `global_db_task.go`, `migrate_workspace.go`, and `session_db.go`; duplication is concentrated in `globaldb` helpers.
- Existing tests already assert that network audit/message timestamp parse failures must be surfaced with wrapped context; session environment `LastSyncAt` parsing lacks an equivalent guard.
- `globaldb.scanSessionEnvironment` now follows the same persisted-timestamp corruption rule and fails scans on malformed `environment_last_sync_at` values.
- `globaldb.ReplaceBridgeInstances` materially improved after reusing prepared bridge-instance payloads in the replacement transaction rather than normalizing/encoding them twice.
- Final validation succeeded with `make verify`, and package coverage stayed above the task target (`internal/store` 84.0%, `globaldb` 80.6%, `sessiondb` 83.0%).

## Files / Surfaces
- `internal/store/sql_helpers.go`
- `internal/store/sqlite.go`
- `internal/store/meta.go`
- `internal/store/types.go`
- `internal/store/sessiondb/session_db.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/perf_bench_test.go`
- `internal/store/globaldb/global_db_session.go`
- `internal/store/globaldb/global_db_bridge.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/store/sessiondb/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/store.md`

## Errors / Corrections
- Initial ADR scan failed because `.compozy/tasks/improvs/adrs/` does not exist in this workspace; corrected by treating ADR context as absent instead of guessing a different path.
- Initial `go test -bench=. -benchmem -count=5 ./internal/store/...` run failed in `BenchmarkReplaceBridgeInstances` because the benchmark fixture constructed disabled bridge instances with non-disabled statuses; corrected the fixture instead of weakening validation.
- The first verification wrapper used zsh's read-only `status` variable and failed before `make verify`; corrected the wrapper and reran the full gate.
- The first clean `make verify` attempt still exposed a `funlen` overage and an unused bridge upsert wrapper; fixed both in `internal/store/` and reran the full gate to green.

## Ready for Next Run
- Task implementation is complete; next run should only need this task memory for historical context or staged/unstaged artifact review.

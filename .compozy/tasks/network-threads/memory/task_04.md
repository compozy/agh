# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 04: SQLite conversation schema/store DTO foundation for network threads/direct rooms/work.
- Scope is schema, migration, DTOs, validation, and tests only; task_05 owns same-transaction message write orchestration.

## Important Decisions
- Use global schema migration version 17 because `globalSchemaMigrations` currently ends at version 16.
- Keep existing runtime-facing `WriteNetworkMessage`/`ListNetworkMessages` method names compiling, but hard-cut their persisted shape to `surface`, `thread_id`, `direct_id`, and `work_id`; do not keep `interaction_id` columns or fallback readers.
- Build store-level validation independently of `internal/network` to preserve the `internal/store/globaldb` boundary that forbids importing `internal/network`.
- Keep migration 1 away from final conversation indexes/tables and let migration 17 create/ensure the final conversation schema, so legacy databases with stale `network_audit_log` or `network_timeline_log` do not fail before the rebuild migration runs.

## Learnings
- Shared memory says task_02 completed the runtime hard cut and task_03 completed direct-room/work primitives; store work should align to those names and invariants.
- Current pre-change schema still has `network_timeline_log.interaction_id` plus old flat timeline indexes, and lacks `network_threads`, `network_direct_rooms`, and `network_work`.
- Wider store tests caught a migration-order bug: final conversation indexes in the v1 schema statement list referenced `surface` before v17 could rebuild old tables. Moving final conversation DDL/index creation to v17 fixed the legacy path while preserving fresh DB final schema after all migrations.

## Files / Surfaces
- Touched: `internal/store/types.go`, `internal/store/network_conversation_types_test.go`, `internal/store/globaldb/global_db.go`, `global_db_network_messages.go`, `global_db_network_audit.go`, network audit normalization, globaldb migration/message/audit tests, and schema migration count tests.

## Errors / Corrections
- Initial test-convention command path from the skill was not present at repo `scripts/check-test-conventions.py`; actual scanner is `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`.
- Existing legacy test files still trip the scanner for pre-existing inline tests; the two new test files pass the scanner. Full package/race verification is the stronger gate for minimally touched legacy files.
- `go test ./internal/store/... ./internal/network ./internal/api/core -count=1` failed until migration 1 stopped creating final conversation indexes before migration 17.
- First full `make verify` failed on task-local lint issues (`funlen` in `scanNetworkMessage`, `lll` in `network_work` DDL). Both were fixed; a rerun passed.

## Ready for Next Run
- Current implementation passes `go test ./internal/store/... ./internal/network ./internal/api/core -count=1` and `go test -race ./internal/store/... ./internal/network ./internal/api/core -count=1`.
- Full `make verify` passed after lint fixes with `DONE 8139 tests` and `OK: all package boundaries respected`.
- Task tracking is marked completed; next step is the final post-tracking `make verify`, self-review, and task-scoped local commit.

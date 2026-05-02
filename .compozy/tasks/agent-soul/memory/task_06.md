# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement migration v13 and GlobalDB persistence for Heartbeat policy snapshots, managed revisions, metadata-only session health, wake state, and wake event audit.
- Success evidence must include fresh DB/reopen/constraint/retention/no-queue tests, focused Go checks, full `make verify`, task tracking updates, and one local commit.

## Important Decisions
- Follow `_techspec_heartbeat.md` DDL and Soul storage conventions: domain persistence types live in `internal/heartbeat`; `internal/store/globaldb` owns migration registration and SQL methods.
- Keep session health storage metadata-only and separate from authored `HEARTBEAT.md`; no task-run lease, claim owner, claim token, or queue columns are added.
- Wake state/events use closed enums for result/source/reason; wake events are retained by explicit `expires_at` cleanup, not converted into runnable work.

## Learnings
- Baseline before code edits: global DB applies through v12 only and `TestGlobalDBSoulMigration` still asserts Heartbeat tables do not exist.
- Task 05 already provides `heartbeat.ResolvedPolicy`, config provenance, prompt contribution, status data, preferences, and diagnostics suitable for snapshot JSON envelopes.
- Implemented Heartbeat storage as migration v13 `add_agent_heartbeat_storage` after Soul v12, with no task-run claim/lease/queue columns in Heartbeat tables.
- Store methods now cover snapshot/revision persistence, session health upsert/list/restart/stale paths, wake state, wake event append/list/read, and bounded wake event retention.
- Focused evidence so far: `go test ./internal/heartbeat ./internal/store/globaldb -count=1`, `go test -race ./internal/heartbeat ./internal/store/globaldb -count=1`, focused `golangci-lint`, and AGH test convention helper all pass.
- Coverage evidence so far: `internal/heartbeat` package is 80.1%; `internal/store/globaldb/global_db_heartbeat.go` is 80.0% and `migrate_heartbeat.go` is 100%. Whole `internal/store/globaldb` remains 78.1% due pre-existing same-package surfaces outside this task.
- Full pre-tracking `make verify` passed with `DONE 7572 tests` and package boundaries OK; non-blocking existing warnings were Node `NO_COLOR`/`FORCE_COLOR`, Vite chunk size, and macOS linker `-bind_at_load`.
- Created local code commit `8096a2fc feat: persist heartbeat storage` with only implementation/test files staged.
- Post-commit `make verify` passed with `DONE 7572 tests` and package boundaries OK.

## Files / Surfaces
- Touched: `internal/heartbeat/persistence.go`, `internal/heartbeat/persistence_test.go`, `internal/store/globaldb/global_db.go`, `internal/store/globaldb/migrate_heartbeat.go`, `internal/store/globaldb/global_db_heartbeat.go`, `internal/store/globaldb/global_db_heartbeat_test.go`, `internal/store/globaldb/global_db_test.go`, `internal/store/globaldb/global_db_soul_test.go`, task tracking/memory files.

## Errors / Corrections
- Fixed compile/lint issues found during focused verification: package helper name collisions, migration `funlen`, repeated ORDER BY literal, `golines` formatting, and context parameter ordering in a test helper.

## Ready for Next Run
- Task implementation, verification, tracking, shared-memory promotion, and local code commit are complete. Final response remains pending.

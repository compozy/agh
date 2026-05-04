# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 07 schema/type foundation for task-run claim leases: canonical run fields, durable SQLite columns/indexes, capability side tables, read-model redaction, and tests. Do not implement `ClaimNextRun` behavior in this task.

## Important Decisions
- Treat the broad `_techspec.md` `ClaimNextRun` wording as future Task 08 behavior; Task 07 only prepares schema, types, and read surfaces.
- Preserve existing `session_id`, claimed actor/origin, and queued/claimed timestamps as ownership/provenance; do not add duplicate owner fields.
- Keep raw `claim_token` available only to internal/synchronous claim paths; public read DTOs expose hash/lease state, never the raw token.

## Learnings
- Contract/OpenAPI/web generated surfaces already contain autonomy-era placeholder fields for task run lease state, but `internal/task.Run` and `task_runs` schema do not yet persist or populate them.
- Baseline `task_runs` has no `claim_token`, `claim_token_hash`, `lease_until`, `heartbeat_at`, `coordination_channel_id`, or capability side tables.
- Task-run read DTO structs already carried safe lease fields, so codegen/web contract files did not need regeneration for Task 07; the missing work was population/redaction.
- Existing global DB migrations are integrity-recorded, so the lease schema is added as migration v7 instead of rewriting historical v1 table DDL.

## Files / Surfaces
- Code surfaces touched: `internal/task`, `internal/store/globaldb`, `internal/api/core`.
- Hook surface touched through existing typed bridge population in `internal/task/manager.go`; no `internal/hooks` type changes were needed.
- Expected tracking surfaces: `.compozy/tasks/autonomous/task_07.md`, `.compozy/tasks/autonomous/_tasks.md`, workflow memory.

## Errors / Corrections
- SQLite's planner did not choose the capability side-table index for a joined status-filter query, so the schema test now proves exact-match capability filtering on the side tables uses the capability indexes directly.
- Public task-run conversion now drops nested `claim_token` fields from metadata/result instead of replacing values, so read JSON does not expose the raw field name or raw value.
- Final lint corrections split the task-run scanner below the package function-length limit, used an explicit aliased SELECT list for idempotency lookup to satisfy gosec SQL construction checks, and simplified claim-token hash prefix handling for staticcheck.

## Ready for Next Run
- Implemented lease fields, coordination channel filtering, capability side tables/indexes, atomic capability persistence, restart reads, and DTO redaction tests.
- Focused checks passed: `go test ./internal/task ./internal/store/globaldb ./internal/api/core`; `go test -tags integration ./internal/store/globaldb ./internal/task`.
- Coverage check: `go test ./internal/task -cover` reports 80.4%. Full `internal/store/globaldb` package remains below 80 because of broader pre-existing package scope, but Task 07 store mapping paths are covered by schema, round-trip, capability, and restart tests.
- Full gate passed after final changes: `make verify` at 2026-04-26 05:12:21 -03 with exit code 0, `0 issues`, `DONE 6124 tests in 55.660s`, and `OK: all package boundaries respected`.
- Local code/test commit created: `3e0574d4e830a69b4de4297c96ef3a492376bc79` (`feat: add task run claim lease schema`). A post-commit `make verify` at 2026-04-26 05:15:20 -03 also exited 0 with `0 issues`, `DONE 6124 tests in 6.497s`, and package boundary checks OK.

# Task Memory: free-iter-004

## Objective Snapshot

- Add global DB orchestration/profile schema migration for task projection fields, run summaries/provenance, and task execution profile selector tables.
- The slice was selected in state iteration 004.

## Important Decisions

- Use numbered global DB migration v17: `add_task_orchestration_profile_schema`.
- Keep queryable orchestration/profile state in typed columns and selector side tables, not `metadata_json`.
- Update the fresh global schema and the migration path together so new DBs and existing v16 DBs converge on the same schema.
- Preserve the legacy task-table rebuild path by adding explicit fallback expressions for the new `tasks` projection columns.

## Learnings

- Existing legacy event-sequence migration tests manually seed migration records; when a new migration assumes prior tables, those fixtures must represent the prior schema accurately.
- A direct full-package `go test -race ./internal/store/globaldb -count=1` can saturate existing 10s test contexts under high parallel SQLite boot load. Isolated race coverage plus the repository `make verify` gate passed after the schema fixes.
- `scripts/check-test-conventions.py` is still absent from this repository; `rg` found no matching helper.

## Files / Surfaces

- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/migrate_task_orchestration_profile.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/store/globaldb/global_db_task_orchestration_schema_test.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/store/globaldb/global_db_soul_test.go`
- `internal/store/globaldb/global_db_heartbeat_test.go`

## Errors / Corrections

- Initial compile failed because the new migration redeclared `migrationColumnSpec`; fixed by reusing the existing package type.
- The legacy task-table rebuild path failed because `tasks_new` and its copy statement did not include new projection columns; fixed with explicit columns and fallback expressions.
- A task-events legacy fixture marked old migrations as applied without creating `task_runs`; fixed the fixture instead of weakening the migration.

## Ready for Next Run

- Next slice can build store/domain models and validation on top of the v17 schema, or continue orchestration hardening with task-service projection mutations.
- Contract, web, site docs, lessons, QA pair, and CodeRabbit rounds remain pending.

## Slice Picked

Add global DB orchestration/profile schema migration for task projection fields, run summaries/provenance, and task execution profile selector tables.

## Acceptance Mapping

- Advances aggregate implementation step 2: orchestration hardening migrations and task-service transition foundations.
- Advances aggregate implementation step 3: `TaskExecutionProfile` migrations and selector-table storage foundation.
- Advances orchestration child build order step 1: schema migrations and store models foundation.
- Does not complete store CRUD, service validation, worker/session profile resolution, claim filtering, APIs, built-in tools, web/docs, QA, or review rounds.

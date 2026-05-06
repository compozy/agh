# Task Memory: free-iter-006

## Objective Snapshot

- Unify task orchestration profile schema DDL between fresh global DB creation and migration v17 to prevent drift.
- The slice was selected in state iteration 006.

## Important Decisions

- Move the task orchestration profile table/index DDL into `schema_task_orchestration_profile.go` so migration v17 and fresh global DB boot consume the same statements.
- Keep `idx_tasks_current_run` behind a named `taskCurrentRunIndexStatement` primitive instead of using the brittle `taskTableIndexStatements[8]` positional reference.
- Compose `globalSchemaStatements` from slice groups with a short `appendSchemaStatements` helper so shared DDL stays declarative without violating `funlen`.

## Learnings

- A direct builder function around the whole fresh global schema violates `golangci-lint` `funlen`; use static slice composition for large schema declarations.
- Fresh schema should include `idx_tasks_current_run` exactly once, through the same v17 statement list used by migrated DBs.

## Files / Surfaces

- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/migrate_task_orchestration_profile.go`
- `internal/store/globaldb/schema_task_orchestration_profile.go`
- `internal/store/globaldb/global_db_task_orchestration_schema_test.go`

## Errors / Corrections

- Initial `make verify` failed at `golangci-lint` with `funlen` on `buildGlobalSchemaStatements`.
- Replaced the long builder with `appendSchemaStatements(groups ...[]string)` and kept DDL in slice literals; no lint suppressions or schema behavior weakening.

## Ready for Next Run

- The schema drift risk called out by the review is closed for task orchestration profile DDL.
- Next slice should continue substantive implementation: store/domain models, task-service projection mutations, `internal/notifications`, review-gate migration, or another small backend foundation slice from the TechSpecs.

## Slice Picked

Unify task orchestration profile schema DDL between fresh global DB creation and migration v17 to prevent drift.

## Acceptance Mapping

- Advances aggregate implementation step 2 by hardening the orchestration migration foundation.
- Advances aggregate implementation step 3 by ensuring the `TaskExecutionProfile` schema foundation is shared between fresh and migrated DB paths.
- Addresses the reviewer-identified schema drift issue around `migrate_task_orchestration_profile.go` and `taskTableIndexStatements[8]`.
- Does not complete store CRUD, task-service transitions, API/contract surfaces, web/docs, QA, or CodeRabbit rounds.

## Verification

- `go test ./internal/store/globaldb -run 'TestGlobalDBTaskOrchestrationProfileSchemaMigration|TestTaskOrchestrationProfileSchemaStatements' -count=1` passed.
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBTaskOrchestrationProfileSchemaMigration|TestTaskOrchestrationProfileSchemaStatements' -count=1` passed.
- `go test ./internal/store/globaldb -count=1` passed.
- `make verify` passed after the lint correction: Bun lint/typecheck/tests passed, Vitest reported 329 files / 2088 tests passed, web build completed, `golangci-lint` reported 0 issues, Go test gate reported `DONE 8127 tests in 165.683s`, and package boundaries were respected.

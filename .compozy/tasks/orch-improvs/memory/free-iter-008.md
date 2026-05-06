# Task Memory: free-iter-008

## Objective Snapshot

- Add global DB review-gate schema migration for task review policy fields, `task_run_reviews`, and `task_runs` review trigger/continuation columns.
- The slice was selected in state iteration 008.

## Important Decisions

- Keep this slice limited to `internal/store/globaldb` schema foundation; task-service review authority, routing, tools, transports, web, and docs remain separate Phase B slices.
- Introduce migration v18, `add_task_review_gate_schema`, instead of widening tables through boot reconciliation.
- Put the review-gate table/index DDL behind shared helpers so fresh DB boot and migration v18 consume the same statements.
- Preserve review policy columns through the legacy `tasks` table rebuild path to avoid future schema loss when old task tables are rebuilt.

## Learnings

- Migration fixtures that mark v17 as applied must manually model a v17-shaped schema; seeding from current `globalSchemaStatements` would incorrectly include v18 columns through migration v1.
- Existing exact task-schema tests are useful drift guards and must be updated when a new schema contract intentionally adds columns.

## Files / Surfaces

- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/migrate_task_review_gate.go`
- `internal/store/globaldb/schema_task_review_gate.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/store/globaldb/global_db_review_gate_schema_test.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/store/globaldb/global_db_task_orchestration_schema_test.go`

## Errors / Corrections

- `go test ./internal/store/globaldb -count=1` initially failed because `TestOpenGlobalDBCreatesTaskSchemaAndIndexes` still expected the pre-review-gate `tasks` columns.
- Updated the existing schema guard to include the intentional review-gate columns and indexes; no production behavior or assertions were weakened.

## Ready for Next Run

- The review-gate persistence foundation is in place for subsequent `internal/task` service work.
- Next slices still need typed task-service review/profile APIs, continuation-run lineage, `internal/notifications`, bridges, bundled skills, API/contract/CLI/tool/web/site/docs, QA, and CodeRabbit gates.

## Slice Picked

Add global DB review-gate schema migration for task review policy fields, `task_run_reviews`, and `task_runs` review trigger/continuation columns.

## Acceptance Mapping

- Advances aggregate implementation step 9 by adding the numbered review-gate migration foundation.
- Implements the `_techspec_review_gate.md` data-model requirements for task policy columns, task run review requests, continuation lineage, and the `task_run_reviews` table.
- Adds fresh-DB and migrated-DB tests for the review-gate schema.
- Does not complete task-service review methods, hooks, transport surfaces, native tools, frontend, docs, QA, or CodeRabbit rounds.

## Verification

- `go test ./internal/store/globaldb -run 'TestGlobalDBReviewGateSchemaMigration|TestReviewGateSchemaStatements|TestGlobalDBTaskOrchestrationProfileSchemaMigration|TestOpenGlobalDBRecordsSchemaMigrationAndRepeatedBootIsIdempotent' -count=1` passed.
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBReviewGateSchemaMigration|TestReviewGateSchemaStatements' -count=1` passed.
- `go test ./internal/store/globaldb -count=1` passed after updating the exact task-schema guard.
- `go test -race ./internal/store/globaldb -count=1` passed in 69.794s.
- `make verify` passed: Bun lint/typecheck/tests passed, Vitest reported 329 files / 2088 tests passed, web build completed, `golangci-lint` reported 0 issues, Go test gate reported `DONE 8132 tests in 121.436s`, and package boundaries were respected.

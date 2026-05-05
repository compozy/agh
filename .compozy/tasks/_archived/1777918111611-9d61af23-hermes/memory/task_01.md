# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 01 persistence/retry foundations: reusable SQLite migration runner, `schema_migrations` records for global/session DBs, migration-backed initialization tests, shared context-aware retry/backoff package, and web/packages-site impact assessment.

## Important Decisions

- Migration runner API lives in `internal/store` as `RunMigrations`, `Migration`, and `AppliedMigrations`; each migration is applied once in version order, under a transaction, with checksum/name integrity checks.
- Global/session DBs each use one v1 create-schema migration list near their owning schema statements.
- Existing global DB pre-run normalization remains ahead of the runner so current legacy-fixture tests still reach the canonical schema; `schema_migrations` records v1 after normalization.
- Added `internal/retry` for shared context-aware jittered exponential backoff. `internal/bridgesdk.RetryDo` now uses the shared wait/delay primitives while keeping bridge-specific error classification.
- Web/packages-site impact is not applicable for code/docs in this task: no public API, generated OpenAPI types, settings pages, examples, or story payloads changed.

## Learnings

- Baseline code search found no existing `schema_migrations`, generic `Migration` runner, or `internal/retry` package.
- Worktree has unrelated pre-existing docs/design and `packages/site` changes; Task 01 implementation must avoid touching them unless impact analysis proves a required change.
- Targeted tests initially failed when global legacy normalizers were removed from the open path; keeping the existing normalization before the new runner preserves current behavior and allows the runner to record canonical schema state.
- `web/` and `packages/site` references to retry/migration are automation query defaults, docs examples, or protocol prose; none consume the new internal Go retry package or SQLite schema table.
- Affected-package coverage initially missed the 80% target; added focused tests for migration validation/failure branches, retry cancellation/defaults, bridge retry cancellation, global migration integrity mismatch, and existing session-liveness helper coverage.
- Fresh final verification on 2026-04-24: `make verify` exited 0 with 5,773 Go tests and package-boundary checks passing; `oxlint` reported 0 warnings/0 errors and `golangci-lint` reported 0 issues. Non-blocking pre-existing toolchain/build warnings remained: Node `NO_COLOR`/`FORCE_COLOR`, Vite chunk-size notice, and macOS linker `-bind_at_load`.

## Files / Surfaces

- Planned surfaces: `internal/store`, `internal/store/globaldb`, `internal/store/sessiondb`, new `internal/retry`, and Task 01 tracking/memory.
- Touched implementation/test surfaces: `internal/store/schema.go`, `internal/store/schema_test.go`, `internal/store/globaldb/global_db.go`, `internal/store/globaldb/migrate_workspace.go`, `internal/store/globaldb/global_db_test.go`, `internal/store/sessiondb/session_db.go`, `internal/store/sessiondb/session_db_test.go`, `internal/retry/retry.go`, `internal/retry/retry_test.go`, `internal/bridgesdk/errors.go`, `internal/bridgesdk/errors_test.go`.
- Additional coverage-support surface: `internal/store/session_liveness_test.go`.

## Errors / Corrections

- Corrected the initial global migration wiring after targeted tests showed old fixture migrations still depend on existing pre-run normalizers.
- Corrected literal nil-context test calls after `make verify` exposed staticcheck runner warnings; tests still cover nil-context guards through helper-returned nil contexts.

## Ready for Next Run

- Task tracking is updated and scoped implementation commit was created: `211f24fb feat: add store migrations and retry foundations`.
- Tracking/memory files are intentionally not included in the implementation commit per workflow staging rules. Current verification evidence:
  - Post-commit `go test -cover ./internal/store ./internal/store/globaldb ./internal/store/sessiondb ./internal/retry ./internal/bridgesdk`
  - `make verify`

# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `sessions.provider` to the global SQLite schema and all migration paths, then make global session register/list/scan/reconcile persist `provider`.
- Repair inactive legacy blank-provider session metadata exactly once before resume or observer reconcile continues, persist the repaired provider immediately, and fail explicitly if the stored agent/provider no longer resolves.
- Finish with provider-aware tests, workflow/task tracking updates, and a clean `make verify`.

## Important Decisions
- Reused shared resolution helpers in `internal/session/manager_workspace.go` so resume-time repair and observer reconcile resolve stored agent/provider state through the same path.
- Observer reconcile now calls `session.RepairLegacyProvider(...)` before normalizing metadata into `store.SessionInfo`, so the global index stops preserving blank providers after the first repair pass.
- Kept legacy handling one-shot only: after repair succeeds, the provider is written back to `meta.json` and later reads use ordinary persisted state with no ongoing blank-provider fallback.

## Learnings
- `loadLegacySessions()` must tolerate both pre-provider schemas and partially migrated schemas, so it now detects the column dynamically and selects `COALESCE(provider, '')`.
- The migration risk boundary is split across two stores: on-disk session metadata and the global `sessions` index. Reconcile has to repair metadata first, then index the repaired provider, or the two layers drift.
- `internal/store/globaldb` package coverage was stuck at `79.9%`; adding a reconcile test for duplicate IDs and zero timestamps raised it to `80.0%` without expanding scope outside task_03 behavior.

## Files / Surfaces
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/store/globaldb/global_db_session.go`
- `internal/store/globaldb/global_db_session_test.go`
- `internal/store/globaldb/global_db_test.go`
- `internal/store/globaldb/global_db_extra_test.go`
- `internal/session/manager_workspace.go`
- `internal/session/resume_repair.go`
- `internal/session/provider_lifecycle_test.go`
- `internal/session/provider_lifecycle_integration_test.go`
- `internal/observe/reconcile.go`
- `internal/observe/reconcile_test.go`
- `internal/observe/helpers_test.go`
- `internal/observe/observer_test.go`
- `internal/extension/host_api_test.go`

## Errors / Corrections
- The task prompt was issued from the daemon-web-ui worktree, but the implementation surface for task_03 lives in `/Users/pedronauck/Dev/compozy/agh`; execution was moved there before editing code.
- After the first round of task-specific tests, `go test -cover ./internal/store/globaldb ./internal/session` reported `internal/store/globaldb` at `79.9%`. Added a narrowly scoped reconcile test instead of weakening assertions or padding unrelated code.
- The first `make verify` failed on a repo-gate `internal/extension` race cleanup issue. Root cause was the host API test harness stopping the original session manager even when `useSessionsWithoutObserver()` had replaced `env.sessions`; cleanup now follows the active manager and repeated race runs passed.

## Ready for Next Run
- Focused verification is green:
- `go test ./internal/store/globaldb ./internal/session ./internal/observe -count=1`
- `go test -tags integration ./internal/session -run 'TestManagerIntegrationProviderPersistsAcrossCreateStatusListAndResume|TestManagerIntegrationLegacyProviderRepairPersistsAndResumeStaysDeterministic' -count=1`
- `go test -cover ./internal/store/globaldb ./internal/session`
- Full repo verification is now also green:
- `make verify`
- Local code-only commit created: `66d69ba1` (`feat: migrate session provider into global index`).
- Post-commit verification also passed on the committed tree:
- `make verify` -> `DONE 5582 tests in 5.025s`
- `OK: all package boundaries respected`

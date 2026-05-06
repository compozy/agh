# Task Memory: free-iter-010

## Objective Snapshot

- Slice: Maintain `tasks.current_run_id` projection through task-run claim, lease release, terminal, and recovery transitions.
- Acceptance mapping: advances the orchestration state projection invariant from `_techspec.md`, the ADR-005 transition matrix for task run lifecycle updates, and the task-service test strategy for active-run read models.

## Important Decisions

- Treat `tasks.current_run_id` as a read projection on `task.Task` and `task.Summary`; callers do not get a public write API.
- Update the projection in the same SQLite transaction as each run transition that owns active-run state: claim, complete lease, fail lease, release lease, recovery, and service-managed `UpdateTaskRun` transitions.
- Reject attempts to project a second active run for the same task instead of overwriting the current projection.
- Keep empty-fresh global DB boot on the numbered migration path, but execute that path in one transaction to prevent repeated migration transaction overhead under race-enabled package tests.
- Snapshot extension lifecycle fields under `Manager.mu` before shutdown/recovery goroutines read them; do not read mutable `managedExtension` fields outside the manager lock.

## Learnings

- Default-parallel `go test -race ./internal/store/globaldb` can saturate the package beyond the repo gate's `-parallel=4`; validate against the actual `make verify` parallelism before changing unrelated test policy.
- Fresh global DB boot was paying many separate migration transactions for empty databases; after new schema slices, that was enough to trip 10s test contexts under package concurrency.
- `internal/extension` had a real race between `Stop` reading `managedExtension.process` and recovery disabling the same extension. The race detector correctly blocked the gate.

## Files / Surfaces

- `internal/task/types.go`
- `internal/task/manager.go`
- `internal/store/schema.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_aux.go`
- `internal/store/globaldb/global_db_task_claim.go`
- `internal/store/globaldb/global_db_task_projection.go`
- `internal/store/globaldb/global_db_task_claim_test.go`
- `internal/store/globaldb/global_db_task_test.go`
- `internal/extension/manager.go`

## Errors / Corrections

- `make verify` first caught `funlen`, `context-as-argument`, and `lll`; the code was refactored instead of suppressing lint.
- A broad default-parallel `globaldb` race run exceeded test contexts, but `go test -race -parallel=4 ./internal/store/globaldb -count=1` matched the gate and passed.
- The first `make verify` run after the projection work failed on SQLite migration timeouts in packages opening fresh global DBs concurrently; the fresh DB bootstrap now applies the numbered global migrations in one transaction.
- The next `make verify` run exposed a real `internal/extension` data race; shutdown now snapshots mutable extension fields under lock, and runtime bridge IDs are read from a locked runtime snapshot.

## Verification

- `go test ./internal/store/globaldb -run 'TestGlobalDBTaskCurrentRunProjection|TestGlobalDBClaimNextRunConcurrentSingleWinner|TestGlobalDBRecoverExpiredRunLeasesThenClaim' -count=1`
- `go test ./internal/task -run 'TestValidateImmutableTaskFields|TestManagerEnrichesTaskSummary|TestManagerReleaseSessionRunLeasesRequeuesActiveRunsStructurally' -count=1`
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBTaskCurrentRunProjection' -count=1`
- `go test ./internal/store/globaldb -count=1`
- `go test ./internal/task -count=1`
- `go test -race -p 1 -parallel 1 ./internal/store/globaldb -count=1`
- `go test -race -parallel=4 ./internal/store/globaldb -count=1`
- `go test -race ./internal/automation -run TestAutomationJobResourceApplyFailurePreservesPreviousRuntime -count=5`
- `go test -race ./internal/soul -run TestManagedSoulAuthoringServiceDeleteRollbackAndHistory -count=3`
- `go test -race ./internal/network -run TestManagerStatusTracksWorkflowMetricsAndStructuredLogs -count=20`
- `go test -race ./internal/extension -run TestManagerDisablesExtensionAfterConsecutiveFailures -count=20`
- `go test -race -parallel=4 ./internal/extension -count=1`
- `make verify`

## Ready for Next Run

- Remaining backend slices still include `TaskExecutionProfile` CRUD/service authority, review service methods, durable notifications, bridge terminal delivery, bundled orchestration skills, API/CLI/native tools, web/site docs, `docs/_memory` lessons, QA pair, and CodeRabbit rounds.

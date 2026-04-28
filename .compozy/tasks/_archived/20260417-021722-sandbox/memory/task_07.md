# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add daemon boot reconciliation for persisted non-local session sandboxs so restart can reattach recoverable sandboxes, discover partial creates by daemon-owned sandbox ID, destroy unrecoverable remote sandboxes, and keep boot non-blocking.
- Completed with targeted daemon tests, integration coverage, focused coverage evidence, and full `make verify`.

## Important Decisions
- Dirty worktree contains prior sandbox task changes, including Daytona provider and existing daemon/environment edits. Treat them as pre-existing work and do not revert.
- Environment restart reconciliation runs in `bootFinalize` after `resourceReconcile.RunBoot` and before `observer.Reconcile`, because `observer.Reconcile` normalizes crashed non-terminal sessions to stopped.
- Added optional `sandbox.Finder` for provider-side lookup by daemon-owned labels before reconciling partial creates.
- Reconciliation does not call `Provider.Prepare` for metadata with empty `InstanceID` unless provider lookup first finds an existing remote instance. This avoids creating a fresh billable sandbox during boot cleanup.
- Task tracking was updated only for task 07. Pre-existing task 05/06 tracking inconsistencies were left untouched.

## Learnings
- TechSpec step 12 requires sandbox reconciliation after `cleanupOrphans` and after canonical resource-runtime boot/reconcile when that runtime is enabled.
- ADR-001 requires remote providers to tag resources with `agh_session_id` and `agh_sandbox_id`; restart cleanup must use daemon-owned identity and provider state for recovery.
- Current `observer.Reconcile` rewrites non-terminal session metadata to `stopped`; sandbox reconciliation must inspect metadata first to satisfy crashed-active reattach behavior.
- Existing `internal/daemon` package coverage remains below 80% package-wide due the size of the package, even with focused new tests. Daytona package coverage is now above 80%.
- Focused coverage for `internal/daemon/sandbox_reconcile.go` is 82.5% statements.

## Files / Surfaces
- Expected surfaces: `internal/daemon/boot.go`, `internal/daemon/orphan.go` patterns, `internal/store/globaldb/global_db_session.go`, `internal/sandbox/registry.go`, `internal/sandbox/types.go`.
- Touched production: `internal/daemon/boot.go`, `internal/daemon/sandbox_reconcile.go`, `internal/sandbox/types.go`, `internal/sandbox/daytona/provider.go`, `internal/sandbox/daytona/state.go`.
- Touched tests: `internal/daemon/sandbox_reconcile_test.go`, `internal/daemon/sandbox_reconcile_integration_test.go`, `internal/sandbox/daytona/provider_test.go`.

## Errors / Corrections
- Corrected initial behavior that would have fallen back to `Prepare` after a partial-create lookup miss. That could create a new remote sandbox on daemon boot; now it logs and skips until an existing `InstanceID` is known.

## Ready for Next Run
- Implementation, tracking, verification, and scoped local commit are complete.

## Verification Evidence
- `go test ./internal/daemon -run TestReconcileDaemonEnvironments -coverprofile=/tmp/daemon-focused.cover -count=1` passed; `sandbox_reconcile.go coverage: 82.5% (184/223 statements)`.
- `go test -tags integration ./internal/daemon -run TestDaemonEnvironmentReconcileIntegration -count=1` passed.
- `go test ./internal/sandbox/daytona -run 'TestDaytonaProviderFindSandboxUsesDaemonEnvironmentLabel|TestDaytonaProviderPrepare' -count=1` passed.
- `go test ./internal/daemon ./internal/sandbox ./internal/sandbox/daytona -count=1` passed.
- `make verify` passed with `DONE 4271 tests in 8.426s` and package boundary verification passed.
- Local commit: `858eda91 feat: reconcile remote sandboxes on daemon boot`.

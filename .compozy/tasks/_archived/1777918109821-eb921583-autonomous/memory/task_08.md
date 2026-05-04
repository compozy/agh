# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the Task 08 claim/lease service on top of Task 07 schema: atomic next-run claims, token hashing/fencing, heartbeat/release/complete/fail lease mutations, expired lease recovery, hook co-emission, and race/recovery tests.

## Important Decisions
- Preserve the existing `ClaimRun(runID)` surface for current operator/internal tests, but make new autonomous next-work flow use `ClaimNextRun(criteria)` and token-fenced mutation methods.
- Raw claim tokens must not be persisted even though the Task 07 schema still has a `claim_token` column; new writes should store only `claim_token_hash` and return raw tokens only from synchronous claim results.
- Use deterministic `Now`/lease duration inputs in task/store APIs and tests; do not use sleeps for lease behavior.
- Keep expired lease recovery as an explicit service/scheduler/boot operation; `ClaimNextRun` claims queued runs and does not silently sweep active expired leases inside the claim transaction.

## Learnings
- Task 07 added lease columns, capability side tables, and safe read DTO fields but intentionally did not implement `ClaimNextRun`.
- Existing hook runtime already has task-run lease events (`lease_extended`, `lease_expired`, `lease_recovered`, `released`), but the task package bridge currently exposes only enqueue/pre-claim/post-claim/recovered.
- Existing manager `CompleteRun`/`FailRun` are unfenced legacy/operator paths; Task 08 needs token-fenced variants for agent claim ownership.
- SQLite treats `ORDER BY 0` as an invalid positional reference; zero-capability claim ordering must use a literal expression such as `(SELECT 0)` instead.
- Full `make verify` exposed an extension manager race where registry-visible disable could precede in-memory hook unregister; fixed production ordering in `internal/extension/manager.go` so disabled extensions no longer transiently expose hooks.

## Files / Surfaces
- Touched code surfaces: `internal/task`, `internal/store/globaldb`, `internal/hooks`, `internal/daemon/task_runtime.go`, and test stubs under `internal/api/testutil`/`internal/daemon`.

## Errors / Corrections
- Focused tests exposed the zero-capability `ClaimNextRun` SQL ordering bug; fixed in production query generation and covered by store claim tests.
- Token fencing checks now run before active-status checks so recovered leases with cleared hashes reject stale holders with `ErrInvalidClaimToken`.
- `make verify` failed once in `internal/extension TestManagerDisablesExtensionAfterConsecutiveFailures`; after fixing disable ordering, `go test ./internal/extension -run TestManagerDisablesExtensionAfterConsecutiveFailures -count=10`, focused task/store/extension tests, and full `make verify` passed.
- Post-commit `make verify` also passed on commit `915ca3ad`.

## Ready for Next Run
- Task 09 can expose agent/operator verbs by calling `ClaimNextRun`, `HeartbeatRunLease`, `CompleteRunLease`, `FailRunLease`, and `ReleaseRunLease`; raw claim tokens should remain only in the synchronous claim response and command inputs.
- Task 11 can call `RecoverExpiredRunLeases` for sweep/boot recovery; it should not claim work directly.

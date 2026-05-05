# ADR-005: Keep `tasks.current_run_id` as a Denormalized Read Projection

## Status

Accepted

## Date

2026-05-05

## Context

AGH currently derives active/current task-run state from `task_runs`. This is the safest source of truth and aligns with archived autonomy decisions. The selected MVP includes `tasks.current_run_id` to improve read paths and task UI/API ergonomics.

The risk is that a task-level pointer could become a second ownership authority if scheduler, coordinator, API, or web code treats it as the source of truth for claimability or terminal state.

## Decision

Keep `tasks.current_run_id` in the MVP as a denormalized read projection only.

The invariant is mandatory:

> `task_runs` remains the only authoritative execution queue and ownership source. `tasks.current_run_id` is a task-service-maintained read projection and must never be used as claim authority, scheduler assignment authority, coordinator ownership authority, or terminal-state authority.

Implementation constraints:

- Only `task.Service`/store transition methods may update `tasks.current_run_id`.
- Updates must happen in the same transaction as the relevant run transition whenever the store supports that transition transactionally.
- Claim, start, complete, fail, release, cancel, recovery, synthetic terminal run creation, and task closure paths must explicitly define how they set or clear the projection.
- Scheduler and coordinator packages may read the projection for display/diagnostics only when useful, but must not use it to claim, assign, or complete work.
- API and web payloads must label/use it as read model state, not as an operation target.
- Projection set/clear helpers must remain unexported inside `internal/task` and callable only from task-service transition methods. There is no public `MaintainCurrentRunID` surface.
- Tests must cover projection updates across successful claims, terminal transitions, recovery, release/requeue, and force-complete/synthetic-terminal paths.

Required transition matrix:

1. `ClaimNextRun` sets the projection to the claimed run.
2. `StartRun` and `AttachRunSession` preserve the projection and fail if it points at another active run.
3. `CompleteRunLease` clears the projection.
4. `FailRunLease` clears the projection.
5. `ReleaseRunLease` and `ReleaseSessionRunLeases` clear the projection before the run can be claimed again.
6. `RecoverExpiredRunLeases` clears the projection for every recovered stale lease.
7. Synthetic terminal run creation sets and clears the projection in the same transaction.
8. Task cancel, archive, delete, and terminal close paths clear the projection.

Synthetic terminal runs are in scope because task-level terminal writes must not discard summary, result, error, or metadata when a task has no active run. A synthetic run is zero-duration, uses the next free attempt number, records the server-derived actor/origin, and is never claimable.

Task-level terminal writes are operator-only and reject when a token-fenced active run exists. Active runs must terminalize through run-level protected task-service paths, not through task-level force-close semantics.

Synthetic run creation must execute inside a write transaction with a no-active-run guard and task-status compare-and-set. A second concurrent task-level terminal request against an already terminal task returns the existing terminal task/run without inserting a duplicate synthetic row or emitting a duplicate terminal event.

## Consequences

### Positive

- Gives the task UI and task read APIs a stable pointer for current-run display.
- Avoids repeated active-run derivation in common read paths.
- Keeps a clear hard boundary between read model and authority.

### Negative

- Adds drift risk if any task-run transition forgets to update the projection.
- Requires extra migration, store tests, and service tests.
- Requires repeated documentation in generated implementation tasks so future work does not misuse it.

### Risks

- A future optimization could accidentally use `current_run_id` to drive claim or coordinator behavior.
- Recovery and release/requeue paths are likely drift points if not tested.
- Force-complete and synthetic-terminal behavior must be specified before implementation to avoid projection ambiguity.

## Rejected Alternatives

### Drop `current_run_id` from MVP

Rejected because the selected scope values the read-model improvement, and the risk is manageable with a strict projection invariant.

### Store projection in a separate `task_runtime_projection` table

Rejected for MVP because it adds schema and implementation complexity without changing the underlying invariant.

### Treat `current_run_id` as authoritative

Rejected because it would create a second task execution authority and violate the existing `task_runs` queue model.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-data-model.md`
- `.compozy/tasks/orch-improvs/analysis/analysis.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-003.md`
- `internal/task/manager.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_claim.go`

# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Make authenticated network peers first-class task-domain writers for create/update/cancel/enqueue flows while preserving server-derived identity/origin, channel-binding enforcement, stale-channel policy, idempotency, and auditability.

## Important Decisions
- `internal/network` owns the peer-authenticated ingress seam, but all task/run mutations still route through `internal/task.TaskManager`.
- Network-originated actor/origin metadata is always derived server-side from `(peer_id, channel)` using `task.DeriveNetworkPeerActorContext`; request IDs are audit-only and do not affect idempotency scope.
- Channel validation is injected into `internal/task.TaskManager` from the daemon (`network.ValidateChannel`) so stale-channel checks can run without introducing a `task -> network` import cycle.
- Stale task bindings block network ingress unless the update explicitly clears or repairs `network_channel`; stale task/run channel snapshots block `StartRun` and `AttachRunSession` before run-state mutation.
- Task-aware network audit entries reuse the existing network audit sinks (`store.NetworkAuditEntry`) via an optional `TaskIngressAuditWriter` extension rather than a parallel audit subsystem.

## Learnings
- `internal/network` already had the right peer registry and audit primitives; the missing piece was a narrow task service seam plus capability-checked peer resolution.
- Origin-scoped task-run idempotency from earlier task work was sufficient for network retries once the network ingress layer reused a stable origin ref (`peer:<id>/channel:<channel>`).
- Stale-channel handling needed to happen in two places: ingress-time repair/rejection for task records and start/attach-time rejection for persisted run snapshots.

## Files / Surfaces
- `internal/network/manager.go`
- `internal/network/tasks.go`
- `internal/network/audit.go`
- `internal/network/tasks_test.go`
- `internal/network/tasks_integration_test.go`
- `internal/network/audit_test.go`
- `internal/task/manager.go`
- `internal/task/errors.go`
- `internal/task/manager_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/task_runtime.go`
- `internal/api/core/errors.go`
- `internal/api/core/errors_test.go`
- `internal/api/core/tasks_test.go`
- `internal/extension/host_api_tasks.go`
- `internal/extension/host_api_test.go`

## Errors / Corrections
- Initial `internal/network/tasks.go` wiring left the task service option and clock helper incomplete; fixed by wiring `TaskService` into `Manager`/`managerOptions`, removing the stubbed helper, and passing action names through peer-resolution rejections so rejected ingress audits have valid kinds.
- Package coverage initially left `internal/network` below the 80% gate; added focused unit coverage for create/cancel paths, ingress validation, and reason mapping.

## Ready for Next Run
- Task implementation is complete and fully verified.
- Evidence:
  - `go test ./internal/task ./internal/network ./internal/api/core ./internal/extension -count=1`
  - `go test -tags integration ./internal/network -count=1`
  - `go test -cover ./internal/task -count=1` (`80.0%`)
  - `go test -cover ./internal/network -count=1` (`81.0%`)
  - `go test -cover ./internal/api/core -count=1` (`80.0%`)
  - `go test -cover ./internal/extension -count=1` (`80.0%`)
  - `make verify`

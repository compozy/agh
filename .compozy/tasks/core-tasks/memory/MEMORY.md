# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- `task_02` added durable `tasks` and `task_runs` persistence to `globaldb`; later task-manager/API work can rely on those store surfaces existing behind `internal/store/globaldb`.
- `task_03` added durable `task_dependencies`, `task_events`, and `task_run_idempotency` persistence to `globaldb`; later task-manager/network/observe work can rely on those store surfaces existing behind `internal/store/globaldb`.
- `task_05` added manager-owned `TaskRun` lifecycle methods (`enqueue`, `claim`, `start`, `attach`, `complete`, `fail`, `cancel`) plus tree cancellation and canonical task reconciliation in `internal/task`; later daemon/API/observe work should call these lifecycle methods instead of mutating run or task status directly.
- `task_06` now boots a daemon-owned `TaskManager` into `RuntimeDeps.Tasks` with an injected session bridge; executable task runs allocate dedicated system sessions by default, explicit attach remains validated in `internal/task`, and daemon boot reconciles persisted `claimed`/`starting`/`running` runs against live session state before servers start.
- `task_07` added shared task/run contracts in `internal/api/contract/tasks.go` plus transport-agnostic core handlers in `internal/api/core/tasks.go`; follow-on HTTP/UDS/CLI work should reuse those contracts/handlers instead of reimplementing task parsing, payload conversion, or task-domain error mapping.
- `task_08` exposed the full task/task-run route inventory through both `internal/api/httpapi` and `internal/api/udsapi`; both server constructors now fail fast without an injected `TaskService`, and daemon transport wiring passes `RuntimeDeps.Tasks` into both factories so follow-on CLI/web work should call those transport routes instead of bypassing the shared API surface.
- `task_09` added the daemon-backed `agh task` CLI surface in `internal/cli`, including task create/list/get/update/cancel, child/dependency commands, and run lifecycle commands; follow-on CLI work should extend `internal/cli/task.go` and the shared `DaemonClient` task/run transport methods instead of bypassing UDS routes.
- `task_10` added explicit automation/task integration without making tasks the universal automation wrapper: automation jobs may opt into `job.task`, direct task-backed jobs materialize/enqueue work through `internal/task`, and linked `automation_runs` persist as `delegated` activation records with `task_id` / `task_run_id`.
- `task_11` added extension Host API task surfaces for list/create/get/update/cancel and task-run lifecycle flows; extension task access is now explicitly capability-gated and routes through the daemon-owned `TaskManager` instead of direct store mutation.
- `task_12` added capability-gated network task ingress in `internal/network`: authenticated remote peers now create/update/cancel tasks and enqueue runs only through the daemon-owned `TaskManager`, task ingress writes are audited through the existing network audit sinks, and stale/mismatched channel bindings are rejected before mutation.
- `task_13` added read-side task observability in `internal/observe`: task summary, metrics, and health views now aggregate durable `tasks`, `task_runs`, `task_events`, and `network_audit_log` data plus live session liveness while keeping observe read-side only.

## Shared Decisions
- `globaldb` task writes preflight missing workspace and task references before insert/update paths so callers get `workspace.ErrWorkspaceNotFound` or `task.ErrTaskNotFound` instead of raw SQLite foreign-key failures.
- `task_runs.session_id` is persisted as nullable text without a sessions foreign key, matching the existing automation-run pattern and allowing run/session audit records to survive independently from live session registry state.
- `TaskEvent` persistence includes immutable origin metadata (`origin_kind`, `origin_ref`) alongside actor metadata so later lifecycle and observe work can audit both who acted and which ingress surface produced the write.
- Task-run idempotency lookup/save is scoped by `(idempotency_key, origin_kind, origin_ref)` instead of a bare key so multi-writer replay protection does not collide across ingress surfaces.
- `GlobalDB.CreateDependency` owns the `BEGIN IMMEDIATE` transaction for dependency edge creation, including duplicate detection, per-task edge-limit enforcement, and cycle rejection under the same SQLite write lock.
- Reverse dependency reconciliation is a first-class store surface via `DependencyStore.ListDependents`; manager code uses it to eagerly recalculate downstream task status after dependency, run, and cancellation changes.
- Task cancellation follows a cooperative-then-forced model through `SessionExecutor`: the manager immediately marks queued/open runs cancelled, requests stop for active runs, and escalates to forced stop after the configured grace period while preserving audit events.
- The daemon-owned task/session bridge reuses the existing session manager surface only from `internal/daemon`; workspace-scoped dedicated task sessions bind by workspace ID, global-scoped task sessions bind by the daemon home path, and boot recovery status mutations still flow through task-manager methods rather than direct store writes.
- Automation-linked agent task creation uses `created_by.kind=agent_session` with `origin.kind=automation` and `origin.ref=run:<automation run id>`; future automation, extension, or network ingress work should preserve that separation between actor identity and ingress origin instead of collapsing both to the same session identity.
- Extension-originated task writes derive `created_by` and immutable `origin` server-side from the trusted extension host context via `task.DeriveExtensionActorContext`; follow-on ingress work should preserve this server-owned identity/origin model and ignore payload-supplied actor metadata.
- Network-originated task writes derive `created_by` and immutable `origin` server-side from the authenticated peer/channel context via `task.DeriveNetworkPeerActorContext`; request IDs remain audit metadata and do not change task-run idempotency scope.
- Adding or changing extension Host API method schemas requires regenerating derived API artifacts with `make codegen` so `openapi/agh.json` and generated SDK contracts stay in sync.
- `internal/task.TaskManager` now supports an injected network-channel validator; daemon wiring passes `network.ValidateChannel`, stale task bindings may only be cleared/repaired through network ingress, and stale task/run channel snapshots are rejected before run start or session attach while recording `task.run_rejected`.
- The daemon-backed CLI integration harness in `internal/cli/cli_integration_test.go` now boots a real task manager and its bridge stub must satisfy `observe.BridgeSource`, including `DeliveryMetrics()`, so future end-to-end CLI task coverage should reuse that fixture shape instead of introducing parallel task-only harnesses.

## Shared Learnings
- Task health now exposes stuck claimed/starting/running runs, queue depth, duplicate-ingress totals, channel-mismatch totals, forced-stop totals, and recovery outcomes through `internal/observe`; transport-layer mapping for those fields remains separate follow-on work if external health APIs need them.

## Open Risks
- No known shared verification blocker after `make verify` passed on 2026-04-14 during `task_05`.

## Handoffs

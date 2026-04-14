# TechSpec Review Round 1: Council Synthesis

**Date**: 2026-04-13
**Reviewers**: Architect, Devil's Advocate, Product Mind, Pragmatic Engineer, Security Advocate
**Verdict**: **Revise before implementation** — core entity design is sound, but the spec over-builds for v1 and has critical gaps in authorization, cancellation propagation, automation overlap, and graph bounds.

---

## Consensus Points (All 5 Agree)

### 1. Task/TaskRun separation is correct
The split between coordination (Task) and execution (TaskRun) is architecturally sound. It preserves session simplicity, supports retries, and creates clean query boundaries. **No advisor challenged ADR-001.**

### 2. The spec over-builds for a greenfield alpha with zero users
Every advisor flagged scope excess — 14+ endpoints, 7+7 status enums, dependency DAGs, multi-writer idempotency, channel binding, and observe projections — all before a single user has created a task. The investment is disproportionate to validated demand.

### 3. Missing authorization model is a blocking gap
The spec defines 4 ingress paths with zero authentication or authorization. `owner_kind/ref`, `created_by_kind/ref`, and `origin_kind/ref` are all self-asserted. Any process on the host can create global tasks, claim runs, and poison audit trails. Workspace scope is a field, not a permission boundary.

---

## Critical Issues (Must Fix)

### C1. Automation duplication — two parallel execution layers
**Raised by**: Devil's Advocate (primary), Architect (secondary)

The existing `internal/automation/` has a `Dispatcher` that already reserves runs, creates sessions, prompts them, and transitions run state. The proposed `TaskManager.ClaimRun → AttachSession → CompleteRun` is the same state machine with different names. The spec never addresses:
- Will automation be rewritten to create Tasks?
- Will `automation_runs` and `task_runs` hold overlapping data?
- Which pipeline starts a session when both are active?

**Recommendation**: The spec must define the relationship between automation dispatch and task runs — either automation becomes a task writer (creating TaskRuns instead of its own runs) or the spec documents explicit non-overlap boundaries.

### C2. No cancellation propagation model
**Raised by**: Architect (primary)

`cancelled` is defined as terminal for both Task and TaskRun, but the spec never describes how cancellation flows through the hierarchy:
- Parent cancelled → are children cancelled?
- Are in-flight runs of cancelled children stopped?
- What about runs attached to sessions?

**Recommendation**: Define cancellation semantics before implementation. This couples task lifecycle to session stop behavior and dependency traversal.

### C3. No authorization model for any surface
**Raised by**: Security Advocate (primary), Architect (secondary)

The HTTP API is unauthenticated by default. UDS is socket-access-only. The spec adds multi-writer ingress from automation, extensions, and network peers with no caller identity verification. Attack scenarios:
- Compromised agent subprocess creates global tasks and hijacks runs
- Workspace-A reader mutates workspace-B tasks via API
- Network peer spoofs `origin_kind: "trusted-agent"`

**Recommendation**: At minimum, define that identity fields are server-derived (not caller-asserted) and that workspace scope enforces access boundaries.

### C4. Unbounded JSON fields — DoS and amplification
**Raised by**: Security Advocate

`metadata_json`, `payload_json`, and `result_json` accept arbitrary JSON with no size limits. A 500MB `result_json` exhausts SQLite page cache and, since it flows into SSE streams, becomes a broadcast amplification attack against all connected web UI clients.

**Recommendation**: State explicit payload size limits (e.g., 64KB per JSON field) in the spec.

---

## High-Priority Issues (Should Fix)

### H1. Dependency graph needs explicit bounds
**Raised by**: Architect, Devil's Advocate

The spec mentions cycle detection but never bounds graph depth or fan-out. SQLite has no recursive CTE constraint checking on INSERT — cycle detection is application-level on every `AddDependency` call. Cross-scope dependencies can span workspaces the caller has no visibility into.

Concurrent edge insertion (A→B and B→A simultaneously) creates a TOCTOU window between check and commit.

**Recommendation**: Define maximum dependency depth (e.g., 32) and max direct dependencies per task (e.g., 64). Document the cycle detection algorithm, not just "central validation."

### H2. Cold-start recovery for orphaned runs
**Raised by**: Devil's Advocate

If the daemon crashes, there is no session stop/crash signal. On restart, `TaskRun.status IN ('claimed','starting','running')` rows have no corresponding live session. The spec mentions reconciliation from session signals but not cold-start sweep.

**Recommendation**: Add a startup reconciliation step in `daemon/boot.go` that marks orphaned active runs as failed.

### H3. Network channel is a dangling reference
**Raised by**: Devil's Advocate, Security Advocate

Channel is a validated string, not a FK. When a channel is removed from network config, tasks bound to it persist with a stale reference. The spec propagates stale channels into new TaskRuns silently. No cleanup path exists.

**Recommendation**: Document what happens to channel-bound tasks when a channel disappears. At minimum, log a warning when propagating a channel that no longer validates.

### H4. TaskEvent duplicates existing audit patterns
**Raised by**: Architect

`TaskEvent` as an immutable audit trail creates a second event-store alongside `sessiondb`. The spec doesn't clarify if this is a thin audit log (like `network_audit_log`) or a full event-sourcing backbone. Two ingestion paths into the `observe` projection layer increases complexity.

**Recommendation**: Cap TaskEvent to audit-only fields. State explicitly that it is NOT event-sourcing.

### H5. SQLite WAL contention at 18+ tables
**Raised by**: Architect

Global DB already has 14+ tables with concurrent writes from automation, bridges, and network. Adding 4 more tables with multi-writer paths increases WAL contention.

**Recommendation**: Mandate `BEGIN IMMEDIATE` transactions for task writes and bounded-interval reconciliation (not synchronous per mutation).

---

## Scope Reduction Recommendations (v1 Slimming)

The council converges on a dramatically smaller v1 that still validates the core hypothesis:

### Cut from v1
| Feature | Reason | Advisor(s) |
|---|---|---|
| Dependency edges + cycle detection | Parent/child hierarchy covers first real workflow. DAG edges are power-user. | Product, Pragmatic, Devil's Advocate |
| Multi-writer ingress adapters (automation, extension, network) | No traffic on those surfaces yet. TaskManager API is sufficient. | Product, Pragmatic |
| Channel binding (ADR-004) | Routing infra for a network feature with zero traffic. | Product, Pragmatic |
| Observe projections + queue health metrics | Dashboards for a system with no operations. | Product, Pragmatic |
| `blocked`/`ready` status + reconciliation engine | Simplify to 3-4 statuses. Derive blocked/ready later when deps exist. | Pragmatic |
| 14+ endpoints → ~6 endpoints | `CreateTask`, `GetTask`, `ListTasks`, `EnqueueRun`, `CompleteRun`, `FailRun` | Devil's Advocate, Pragmatic |
| TaskEvent audit trail | Add in a single migration when someone asks "what happened" | Pragmatic |

### Keep in v1
| Feature | Reason |
|---|---|
| Task + TaskRun as separate entities (ADR-001) | Correct boundary, all agree |
| Global + workspace scope (ADR-002, partial) | Needed for multi-workspace coordination |
| Parent/child hierarchy via `parent_task_id` | Core subtask UX |
| Queue-first run lifecycle (ADR-003) | Enables deferred execution |
| Central TaskManager authority | Correct invariant model |
| Session attachment bridge | Core value: run → session linkage |
| REST + UDS + CLI surface (~6 endpoints) | User-facing value |

### Estimated effort
- **Full spec**: 6-8 person-weeks (Pragmatic Engineer estimate)
- **Slimmed v1**: 2-3 person-weeks
- **Difference**: 4-5 weeks freed for web UI, extension registry, or other features

---

## Missing from Spec (Add Before Implementation)

1. **Authorization model** — who can create/read/mutate tasks at each scope?
2. **Cancellation propagation** — parent→child, task→run→session flow
3. **Automation/task boundary** — how automation dispatch relates to task runs
4. **Cold-start recovery** — daemon restart orphaned-run sweep
5. **Dependency graph bounds** — max depth, max fan-out, detection algorithm
6. **JSON field size limits** — explicit caps on metadata/payload/result
7. **TaskManager interface boundary** — injected `SessionAllocator` interface, not direct `session/` import
8. **Web UI milestone** — backend without UI delivers developer value, not user value

---

## ADR Verdicts

| ADR | Verdict | Notes |
|---|---|---|
| ADR-001 (Task/TaskRun split) | **Keep as-is** | Unanimous approval |
| ADR-002 (Scope + hierarchy + deps) | **Keep scope + hierarchy, defer dependency edges** | Deps are premature for v1 |
| ADR-003 (Queue-first + central authority) | **Keep, simplify statuses** | 3-4 statuses per entity in v1 |
| ADR-004 (Channel binding) | **Defer to v2** | No validated demand |

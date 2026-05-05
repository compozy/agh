# ADR-002: Use Queryable Task-Owned State for Orchestration Hardening

## Status

Accepted

## Date

2026-05-05

## Context

`orch-improvs` needs durable state for handoff summaries, current-run read models, task-level watchdog limits, task execution profiles, spawn-failure circuit breaking, and notification cursors. AGH already stores task and run metadata through explicit task tables plus `metadata_json` and `result_json` payloads.

Opaque JSON is useful for arbitrary agent output, but the orchestration hardening fields are operational state. They need validation, queryability, indexing, predictable migrations, and stable API projections.

The greenfield-alpha policy rejects compatibility shims and boot-time schema patching. Schema changes must use numbered migrations.

## Decision

Use explicit typed columns and side tables for queryable orchestration state.

The TechSpec should include these planned persisted fields:

- `task_runs.summary TEXT NOT NULL DEFAULT ''` for bounded worker/coordinator handoff and terminal-run summaries.
- `tasks.current_run_id TEXT REFERENCES task_runs(id) ON DELETE SET NULL` as a denormalized read projection, governed by ADR-005.
- `tasks.max_runtime_seconds INTEGER NOT NULL DEFAULT 0` for per-task runtime watchdog override, where zero disables the task-specific limit.
- `tasks.spawn_failure_count INTEGER NOT NULL DEFAULT 0` for task-service-owned spawn circuit breaker accounting.
- `tasks.last_spawn_error TEXT NOT NULL DEFAULT ''` for the latest bounded spawn failure summary.
- `task_runs.claimed_agent_name`, `task_runs.claimed_peer_id`, `task_runs.terminalized_by_session_id`, `task_runs.terminalized_by_agent_name`, `task_runs.terminalized_by_peer_id`, `task_runs.terminalized_by_actor_kind`, and `task_runs.terminalized_by_actor_ref` for run provenance and review self-review exclusion.
- `task_execution_profiles` as a task-owned table for coordinator, worker, review, participant, and sandbox profile scalars.
- `task_profile_agents`, `task_profile_channels`, `task_profile_peers`, and `task_profile_capabilities` as selector side tables for matchable profile policy.
- A durable notification cursor table governed by ADR-003.
- `bridge_task_subscriptions` as a bridge-owned side table for terminal task notification targets. Cursor progress for each subscription remains in `notification_cursors`.
- `tasks.review_policy`, `tasks.review_max_rounds`, `tasks.review_round`, `tasks.last_review_id`, `tasks.last_review_outcome`, `tasks.review_circuit_opened_at`, and `tasks.review_circuit_reason` as task-owned review policy and rollup fields.
- `task_run_reviews` as a task-owned side table for review requests, reviewer routing metadata, verdicts, bounded evidence, circuit state, and continuation guidance.
- `task_runs.review_required`, `task_runs.review_request_round`, `task_runs.review_policy_snapshot`, and `task_runs.review_request_id` as typed review-request trigger state for recovery after terminalization.
- `task_runs.parent_run_id`, `task_runs.review_id`, `task_runs.review_round`, `task_runs.continuation_reason`, `task_runs.missing_work_json`, and `task_runs.next_round_guidance` as typed continuation-run lineage and context fields for review-driven follow-up work.

`spawn_failure_count` is scoped only to spawn/session-start failures such as `spawn_failed`, `session_unreachable`, and `provider_auth`. `internal/session.Manager` classifies spawn failures and calls task-service-owned increment methods. The counter resets from successful `AttachRunSession`, not from claim alone. When the configured limit is reached, a task-service-owned transaction opens the circuit and prevents `ClaimNextRun` from returning the task until an operator or later task-service transition clears it.

`metadata_json` and `result_json` remain for opaque payloads only. They must not become the storage location for operational orchestration state that needs query predicates, indexes, or contract-level validation.

Task execution profile selectors are stored in side tables because worker claim eligibility, reviewer routing, dashboard filtering, validation, and generated contract projections need exact-match predicates. Bounded coordinator guidance may remain a text field because it is prompt guidance, not a matching dimension.

`task_run_reviews.missing_work_json` and `task_runs.missing_work_json` are allowed as bounded canonical JSON arrays because missing-work items are ordered continuation guidance, not a SQL matching dimension. Review outcome, status, round, reviewer, deadline, circuit state, review-request trigger state, continuation source, continuation reason, and run provenance remain typed columns.

Migration ownership for review state is pinned to the review-gate migration. It creates `task_run_reviews` and the `task_runs` review trigger/continuation columns that reference `task_run_reviews` in the same numbered migration. The orchestration hardening migration must not add forward FKs to `task_run_reviews`.

All schema changes must be implemented as numbered global DB migrations and covered by fresh-DB and migrated-DB tests. `EnsureSchema`-style reconciliation and compatibility fallback reads are out of scope.

## Consequences

### Positive

- Makes scheduler, dashboard, context bundle, and notifier reads straightforward and indexable.
- Keeps handoff summaries visible without parsing arbitrary result payloads.
- Aligns with AGH's hard-cut migration policy.
- Keeps future task-generation concrete: implementation tasks can name specific columns, indexes, and store methods.
- Lets review gate feed `TaskContextBundle.ReviewContinuation` from persisted task-owned state instead of prompt/channel memory.
- Lets idempotent rejected-review replay find the existing continuation run by `task_runs.review_id` without parsing JSON metadata.
- Lets review-request recovery clear pending `task_runs.review_required` state by linking the terminal run to `task_runs.review_request_id`.

### Negative

- Adds migration and store maintenance work.
- Requires careful transaction design so projections and counters cannot drift from authoritative task-run transitions.
- Requires codegen/API updates when contract payloads expose the new fields.
- Requires strict migration ordering because `task_runs.review_request_id` and `task_runs.review_id` reference `task_run_reviews`.

### Risks

- `current_run_id` could be mistaken for ownership authority if the invariant is not repeated in implementation tasks.
- `spawn_failure_count` could become a generic retry policy if the circuit breaker is not scoped to spawn failures.
- Summary fields can become unbounded prompt dumps unless size limits are enforced consistently.
- Task-specific worker/reviewer/sandbox selection could be hidden in `metadata_json` unless the profile tables and API contracts are treated as the only runtime selection surface.
- Duplicate migration ownership for review columns could produce FK-ordering bugs; task generation must keep review trigger/continuation columns in the review-gate migration only.

## Rejected Alternatives

### Store orchestration state in `metadata_json`

Rejected because the fields are operational and queryable, not arbitrary extension payloads.

### Store task execution profiles in `metadata_json`

Rejected because profiles drive worker eligibility, reviewer routing, session start, and sandbox selection. Those fields require validation, indexes, and generated contract types.

### Store handoff summaries only in channel messages

Rejected because channel messages are coordination transport, not task-run state. Handoff summaries must survive as run/task projections.

### Store review verdicts only in channel messages

Rejected because review verdicts and continuation guidance are task lifecycle inputs. They must survive restart, compaction, channel backlog pruning, and reviewer-session termination.

### Boot-time schema repair

Rejected because AGH greenfield alpha requires hard-cut numbered migrations, not compatibility reconciliation.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-data-model.md`
- `.compozy/tasks/orch-improvs/analysis/analysis.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_task-execution-profile.md`
- `.compozy/tasks/orch-improvs/_techspec_review_gate.md`
- `.compozy/tasks/orch-improvs/adrs/adr-010-task-execution-profiles-are-typed-overlays.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-003.md`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_task.go`
- `internal/store/globaldb/global_db_task_claim.go`

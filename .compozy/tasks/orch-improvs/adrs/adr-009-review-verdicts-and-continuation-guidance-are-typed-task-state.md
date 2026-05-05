# ADR-009: Review Verdicts and Continuation Guidance Are Typed Task State

## Status

Accepted

## Date

2026-05-05

## Context

The Codex Loop `goal` mechanism uses a structured verdict containing completion status, confidence, reason, missing work, and next-round guidance. That shape is valuable because continuation becomes more than "try again"; the next round receives concrete, review-derived guidance.

In AGH, storing this only as prompt prose or channel discussion would be fragile. It would not be queryable, would not survive all recovery paths, and would not support contract-generated web/CLI/API surfaces.

## Decision

Review verdicts and continuation guidance are typed task-owned state.

AGH will persist review requests and verdicts in `task_run_reviews`, with bounded typed fields:

- outcome;
- confidence;
- reason;
- delivery id;
- missing work;
- next-round guidance;
- reviewer identity;
- review evidence;
- review timestamps;
- routing metadata;
- circuit state.

Rejected reviews create continuation runs through task-service-owned transitions. The continuation run stores typed lineage and guidance on `task_runs`:

- `review_required`, `review_request_round`, and `review_policy_snapshot` for terminal-run review-request recovery;
- `review_request_id` linking a terminal run to its durable attempt-1 review request after the follow-up transaction clears `review_required`;
- `parent_run_id`;
- `review_id`;
- `review_round`;
- `continuation_reason`;
- bounded canonical `missing_work_json`;
- `next_round_guidance`.

The next worker receives the latest rejected review through `TaskContextBundle.ReviewContinuation`, assembled from the continuation run's typed lineage and cross-checked against `task_run_reviews`.

Review evidence may include bounded prose, but the typed verdict fields are the contract. Full channel or session transcripts are not stored as review authority.

Reviewer selection fields from `TaskExecutionProfile.Review` may be copied into review route metadata for auditability, but they remain selection inputs. They are not verdict fields and cannot replace `RecordRunReview`.

`delivery_id` is required for verdict submission idempotency. `RecordRunReview` accepts idempotent replay only when `review_id`, `run_id`, actor identity, outcome, and `delivery_id` match the persisted verdict. For rejected verdicts, the same transaction writes the verdict and creates the continuation run; replay returns the existing continuation via `task_runs.review_id = review_id`.

Review request creation is idempotent through unique `(run_id, review_round, attempt)`, with the terminal follow-up path always using `attempt = 1`. Reviewer retry attempts insert new `task_run_reviews` rows with incremented `attempt` only after the previous attempt is terminal. This allows bounded retry history without duplicate attempt-1 request rows.

The review-gate migration owns both `task_run_reviews` and the `task_runs` review trigger/continuation columns that reference it. They must be created in the same numbered migration to avoid FK ordering ambiguity.

## Consequences

- Review state is queryable by task, run, reviewer, status, deadline, and round.
- Web, CLI, UDS, HTTP, hooks, and SSE can present the same review state from generated contracts.
- Context bundles can deterministically include the exact guidance that caused a continuation run.
- Review-table migrations and bounds are required before reviewer routing is useful.
- `task_runs` gains continuation columns and indexes, increasing migration scope but keeping the queue single-source.
- Implementation must distinguish opaque bounded prose (`review_text`) from typed fields (`outcome`, `missing_work`, `next_round_guidance`).
- Implementation must clear pending review-request trigger state and link the terminal run to the durable review request in the same transaction that creates or returns the attempt-1 review row.

## Rejected Alternatives

### Store verdicts in `metadata_json`

Rejected because review outcome, status, round, reviewer, and circuit state are queryable runtime state, not opaque metadata.

### Store missing work as separate SQL rows in MVP

Rejected for MVP because missing work is a bounded ordered guidance list, not a matching/query dimension. A canonical bounded JSON array in `missing_work_json` is sufficient as long as outcome/round/status/reviewer fields remain typed columns.

### Treat reviewer confidence as authority

Rejected because confidence is an audit/debug signal only. The outcome and task-service transition determine behavior.

## References

- [`../_techspec.md`](../_techspec.md)
- [`../_techspec_review_gate.md`](../_techspec_review_gate.md)
- [`adr-010-task-execution-profiles-are-typed-overlays.md`](adr-010-task-execution-profiles-are-typed-overlays.md)
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/goal_confirm.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/store.go`

# ADR-007: Review Gate Is a Post-Terminal Continuation Loop

## Status

Accepted

## Date

2026-05-05

## Context

The orchestration hardening spec already preserves `task_runs` as the durable execution queue and keeps run status transitions narrow: queued, claimed, starting, running, completed, failed, and canceled. The proposed review-gate behavior is inspired by the Codex Loop `goal` mechanism, where an external review checks whether the objective is complete and, when not complete, returns missing work and next-round guidance.

A tempting design is to add a blocking `pending_review` run status between running and completed/failed. That would make review feel like part of the execution state machine, but it would also expand lease release semantics, heartbeat handling, active-run detection, `current_run_id`, max-runtime enforcement, retries, and task terminalization rules.

AGH needs the continuation value of the goal loop without turning review into a second execution lifecycle.

## Decision

Review gate v1 is post-terminal.

Workers terminalize runs through the existing task-service-owned execution transitions. If task review policy applies, `task.Service` creates a persisted attempt-1 review request in a follow-up task-service transaction after the terminal transition commits. That follow-up transaction is idempotent on `(run_id, review_round, attempt = 1)`, writes `task_runs.review_request_id`, clears `task_runs.review_required`, and emits `task.run_review_requested` only after the review row is durable. This makes crash recovery explicit: after a crash between terminal commit and review-request creation, daemon recovery asks `task.Service` for terminal runs with `review_required = 1` and no `review_request_id`, then re-runs the same idempotent request path.

The coordinator is woken through a typed call-site callback wired in `internal/daemon`, not by tailing `task_events` or `task_run_reviews`. The callback is a nudge only; the coordinator reads persisted review state through task-service APIs before routing.

The review verdict decides whether the terminal outcome is accepted or whether a new continuation run is enqueued with typed guidance. `RecordRunReview` persists the verdict, task review rollups, review events, and any rejected-review continuation run in one task-service-owned `BEGIN IMMEDIATE` transaction. Idempotent rejected-verdict replay returns the existing continuation run by `task_runs.review_id = review_id`; it never enqueues a duplicate.

MVP will not add `pending_review` to `task_runs.status`.

Review outcomes:

- `approved`: accept the terminal outcome.
- `rejected`: record missing work and enqueue a continuation run while `max_rounds` remains.
- `blocked`: block the task through task-service-owned state.
- `error`, `timeout`, `invalid_output`: apply bounded retry and circuit policy; they never approve work.

The previously terminal run remains historically terminal. Review state is a separate task-owned layer.

## Consequences

- Existing run lifecycle semantics remain stable.
- Review can be added after orchestration hardening without rewriting lease and heartbeat semantics.
- A completed run can be followed by a continuation run when review rejects the work.
- Task detail/read models must show both execution terminal state and review acceptance state so operators do not confuse "run completed" with "review approved".
- Continuation loops require explicit `max_rounds` and circuit-open behavior to avoid infinite work.
- Implementers must add recovery coverage for the follow-up review-request transaction and atomicity coverage for verdict-plus-continuation enqueue.
- Bridge terminal notifications must not treat review-pending run terminal events as accepted final task notifications. Delivery waits for the accepted final terminal event, such as `task.run_review_approved` for review-gated runs.

## Rejected Alternatives

### Add `pending_review` run status in MVP

Rejected for MVP because it expands the execution state machine and touches every lease, heartbeat, current-run, active-run, timeout, retry, scheduler, and terminalization path.

### Store review as prompt-only guidance

Rejected because continuation guidance must survive restarts, compaction, channel loss, and reviewer-session termination.

### Treat review rejection as rewriting the previous run

Rejected because run history must remain append-only and truthful. A rejected completed run is still a completed execution attempt; it is not an uncompleted run.

## References

- [`../_techspec.md`](../_techspec.md)
- [`../_techspec_review_gate.md`](../_techspec_review_gate.md)
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/goal_confirm.go`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/hooks.go`
